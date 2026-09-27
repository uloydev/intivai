package api

import (
	"context"
	"errors"
	"net"
	"strings"
	"sync"
	"time"

	fiberws "github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/intivai/backend/internal/iam/api"
	"github.com/intivai/backend/internal/iam/application"
	ivapp "github.com/intivai/backend/internal/interview/application"
	"github.com/intivai/backend/internal/interview/application/session"
	ivdomain "github.com/intivai/backend/internal/interview/domain"
	"github.com/intivai/backend/internal/llm"
	sbapp "github.com/intivai/backend/internal/sandbox/application"
	sharederrors "github.com/intivai/backend/internal/shared/errors"
	"github.com/intivai/backend/internal/shared/httpapi"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// tracer resolves the global provider PER CALL — package-level otel.Tracer
// vars freeze onto the first installed provider (global delegation is
// once-only) and silently miss late telemetry.Init calls.
func tracer() trace.Tracer {
	return otel.Tracer("github.com/intivai/backend/internal/interview/api")
}

// readDeadline — per candidate frame (Research §2: PerQuestionTimeout = 3 min).
// Stricter than the domain idle rule — a silent candidate disconnects at 3
// minutes; the 5-minute idle/expiry path still guards server-side state.
const readDeadline = ivdomain.PerQuestionTimeout

// Server heartbeat (Research §2): ping every 30s; drop the socket when the
// client does not answer with a pong within 10s.
const (
	heartbeatInterval = 30 * time.Second
	pongWait          = 10 * time.Second
)

// jsonConn is the outbound surface wsWriter needs from the WS connection
// (satisfied by *fiberws.Conn); an interface keeps the writer unit-testable.
type jsonConn interface {
	WriteJSON(v any) error
}

// wsWriter serializes all outbound frames through one writer goroutine
// (gorilla/websocket allows a single concurrent writer), so the read loop,
// the LLM stream goroutine and the code-run goroutine can all call send safely.
type wsWriter struct {
	ch      chan any
	done    chan struct{}
	mu      sync.Mutex
	closed  bool
	connCtx context.Context
	cancel  context.CancelFunc // owned here so teardown order is structural
}

func newWSWriter(conn jsonConn, connCtx context.Context, cancel context.CancelFunc) *wsWriter {
	w := &wsWriter{ch: make(chan any, 64), done: make(chan struct{}), connCtx: connCtx, cancel: cancel}
	go func() {
		defer close(w.done)
		for frame := range w.ch {
			if err := conn.WriteJSON(frame); err != nil {
				return
			}
		}
	}()
	return w
}

func (w *wsWriter) send(frame any) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed || w.connCtx.Err() != nil {
		return
	}
	select {
	case w.ch <- frame:
	case <-w.connCtx.Done():
	}
}

// sendError sends the error frame for a domain error (machine-readable code).
func (w *wsWriter) sendError(err error) {
	w.send(session.ErrorFrame(err))
}

// close performs the ordered teardown (D18): cancel fires FIRST — a sender
// parked on a full channel holds w.mu and can only wake via connCtx.Done, so
// cancellation must precede the lock acquisition or close deadlocks when the
// writer goroutine has died mid-stream. With the context canceled no new
// sender can pass the flag check either, so the mutex wait below is bounded.
func (w *wsWriter) close() {
	w.cancel()
	w.mu.Lock()
	if !w.closed {
		w.closed = true
		close(w.ch)
	}
	w.mu.Unlock()
	<-w.done
}

// maxConcurrentCodeRuns bounds HOW MANY sandbox runs may execute
// simultaneously across all connections (D24).
const maxConcurrentCodeRuns = session.MaxConcurrentCodeRuns

// maxWSFrameBytes — abuse bound on any single client frame (I13).
const maxWSFrameBytes = 512 * 1024

type ChatHandler struct {
	svc          *ivapp.InterviewService
	llm          llm.Provider
	tokens       application.TokenProvider
	log          zerolog.Logger
	sessions     SessionRegistry
	codeRunner   sbapp.CodeRunner
	runSemaphore chan struct{}
	engine       *session.DialogueEngine
}

func NewChatHandler(svc *ivapp.InterviewService, llmClient llm.Provider, tokens application.TokenProvider, log zerolog.Logger, sessions ...SessionRegistry) *ChatHandler {
	var reg SessionRegistry = NewMemorySessionRegistry()
	if len(sessions) > 0 && sessions[0] != nil {
		reg = sessions[0]
	}
	return &ChatHandler{
		svc:          svc,
		llm:          llmClient,
		tokens:       tokens,
		log:          log,
		sessions:     reg,
		runSemaphore: make(chan struct{}, maxConcurrentCodeRuns),
		engine:       session.NewDialogueEngine(svc, llmClient, log),
	}
}

// WithCodeRunner attaches the sandbox executor (sidecar client) used by the
// WS code.run frame. Set in main; nil keeps code.run disabled with an error
// frame (fail closed when the sandbox sidecar is unavailable).
func (h *ChatHandler) WithCodeRunner(r sbapp.CodeRunner) *ChatHandler {
	h.codeRunner = r
	return h
}

// Create — POST /interviews (recruiter).
func (h *ChatHandler) Create(c *fiber.Ctx) error {
	actor, err := api.RequireActor(c)
	if err != nil {
		return err
	}
	var req struct {
		ApplicationID string `json:"application_id"`
		QuestionCount int    `json:"question_count"`
	}
	if err := c.BodyParser(&req); err != nil {
		return httpapi.Error(c, sharederrors.NewDomainError("INVALID_INPUT", "invalid body"))
	}
	appID, err := uuid.Parse(req.ApplicationID)
	if err != nil {
		return httpapi.Error(c, sharederrors.NewDomainError("INVALID_INPUT", "invalid application_id"))
	}
	result, err := h.svc.CreateInterview(c.UserContext(), actor, ivapp.CreateInterviewCommand{
		ApplicationID: appID, QuestionCount: req.QuestionCount,
	})
	if err != nil {
		return httpapi.Error(c, err)
	}
	return httpapi.Created(c, result)
}

// InvitePreview — GET /api/v1/public/invite-preview?token=<token> (public, pre-flight).
func (h *ChatHandler) InvitePreview(c *fiber.Ctx) error {
	token := strings.TrimSpace(c.Query("token"))
	if token == "" {
		token = strings.TrimSpace(c.Query("t"))
	}
	if token == "" {
		return httpapi.Error(c, sharederrors.NewDomainError("BAD_REQUEST", "invitation token is required"))
	}
	preview, err := h.svc.GetInvitePreview(c.UserContext(), token)
	if err != nil {
		return httpapi.Error(c, err)
	}
	return httpapi.OK(c, preview)
}

// Consent — POST /candidate/interviews/:id/consent (candidate, invitation
// token). Records GDPR consent; the chat refuses to start without it.
func (h *ChatHandler) Consent(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httpapi.Error(c, sharederrors.NewDomainError("INVALID_INPUT", "invalid interview id"))
	}
	var req struct {
		InvitationToken string `json:"invitation_token"`
	}
	if err := c.BodyParser(&req); err != nil {
		return httpapi.Error(c, sharederrors.NewDomainError("INVALID_INPUT", "invalid body"))
	}
	if err := h.svc.GiveConsent(c.UserContext(), id, strings.TrimSpace(req.InvitationToken)); err != nil {
		return httpapi.Error(c, err)
	}
	return httpapi.OK(c, map[string]bool{"consent_given": true})
}

// Ticket — POST /candidate/interviews/:id/ticket (candidate, invitation token).
func (h *ChatHandler) Ticket(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httpapi.Error(c, sharederrors.NewDomainError("INVALID_INPUT", "invalid interview id"))
	}
	var req struct {
		InvitationToken string `json:"invitation_token"`
	}
	if err := c.BodyParser(&req); err != nil {
		return httpapi.Error(c, sharederrors.NewDomainError("INVALID_INPUT", "invalid body"))
	}
	result, err := h.svc.IssueTicket(c.UserContext(), ivapp.IssueTicketCommand{
		InterviewID: id, InvitationToken: strings.TrimSpace(req.InvitationToken),
	})
	if err != nil {
		return httpapi.Error(c, err)
	}
	return httpapi.OK(c, result)
}

// RequestHuman — POST /candidate/interviews/:id/request-human (candidate,
// invitation token). Records the candidate's request for a human interviewer.
func (h *ChatHandler) RequestHuman(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httpapi.Error(c, sharederrors.NewDomainError("INVALID_INPUT", "invalid interview id"))
	}
	var req struct {
		InvitationToken string `json:"invitation_token"`
	}
	if err := c.BodyParser(&req); err != nil {
		return httpapi.Error(c, sharederrors.NewDomainError("INVALID_INPUT", "invalid body"))
	}
	if err := h.svc.RequestHuman(c.UserContext(), id, strings.TrimSpace(req.InvitationToken)); err != nil {
		return httpapi.Error(c, err)
	}
	return httpapi.OK(c, map[string]string{"message": "your request has been noted. a team member will follow up."})
}

// Telemetry — POST /candidate/interviews/:id/telemetry (candidate beacon/HTTP fallback).
func (h *ChatHandler) Telemetry(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httpapi.Error(c, sharederrors.NewDomainError("INVALID_INPUT", "invalid interview id"))
	}
	var req struct {
		InvitationToken string                     `json:"invitation_token"`
		Ticket          string                     `json:"ticket"`
		EventType       string                     `json:"event_type"`
		Timestamp       string                     `json:"timestamp"`
		QuestionIdx     int                        `json:"question_idx"`
		Details         *ivdomain.TelemetryDetails `json:"details"`
	}
	if err := c.BodyParser(&req); err != nil {
		return httpapi.Error(c, sharederrors.NewDomainError("INVALID_INPUT", "invalid body"))
	}

	token := strings.TrimSpace(req.Ticket)
	if token == "" {
		token = strings.TrimSpace(req.InvitationToken)
	}
	if token == "" {
		header := c.Get("Authorization")
		if strings.HasPrefix(header, "Bearer ") {
			token = strings.TrimPrefix(header, "Bearer ")
		}
	}
	if token == "" {
		return httpapi.Error(c, sharederrors.NewDomainError("UNAUTHORIZED", "missing authentication token or ticket"))
	}

	var evTime time.Time
	if req.Timestamp != "" {
		var err error
		evTime, err = time.Parse(time.RFC3339, req.Timestamp)
		if err != nil {
			return httpapi.Error(c, sharederrors.NewDomainError("INVALID_INPUT", "invalid telemetry timestamp"))
		}
	}
	if evTime.IsZero() {
		evTime = time.Now()
	}

	event := ivdomain.ProctoringEvent{
		Type:        ivdomain.ProctoringEventType(req.EventType),
		Timestamp:   evTime,
		QuestionIdx: req.QuestionIdx,
		Details:     req.Details,
	}

	if err := h.svc.RecordCandidateTelemetry(c.UserContext(), id, token, event); err != nil {
		return httpapi.Error(c, err)
	}
	return httpapi.OK(c, map[string]string{"status": "ok"})
}

// RequireTicket — pre-upgrade gate: Bearer must be a ws_ticket bound to this
// interview. Non-upgrade requests get 401; upgrades proceed to Chat.
func (h *ChatHandler) RequireTicket(c *fiber.Ctx) error {
	// Browsers cannot set the Authorization header on a WebSocket — accept
	// ?ticket= as well. Non-browser clients may keep using the header.
	header := c.Get("Authorization")
	token := ""
	if strings.HasPrefix(header, "Bearer ") {
		token = strings.TrimPrefix(header, "Bearer ")
	} else if q := c.Query("ticket"); q != "" {
		token = q
	}
	if token == "" {
		return httpapi.Error(c, sharederrors.NewDomainError("UNAUTHORIZED", "missing ws ticket"))
	}
	claims, err := h.tokens.Parse(token)
	if err != nil || claims.Type != application.TokenTypeWSTicket {
		return httpapi.Error(c, sharederrors.NewDomainError("UNAUTHORIZED", "invalid ws ticket"))
	}
	if claims.Extra.InterviewID != c.Params("id") {
		return httpapi.Error(c, sharederrors.NewDomainError("UNAUTHORIZED", "ticket not bound to this interview"))
	}
	c.Locals("ws_claims", claims)
	// J4: stash the fiber request context — it carries the HTTP server span
	// (httpmw.Tracing). fiberws copies fasthttp user values into conn.Locals
	// BEFORE the upgrade, so the WS handler can seed its connection context
	// from it (see Chat) and keep every interview span on the handshake trace.
	c.Locals("ws_fiber_ctx", c.UserContext())
	return c.Next()
}

func ticketInterviewID(claims *application.Claims) (uuid.UUID, bool) {
	if claims == nil {
		return uuid.Nil, false
	}
	id, err := uuid.Parse(claims.Extra.InterviewID)
	return id, err == nil
}

// Chat — WS /candidate/interviews/:id/chat. Single writer goroutine serializes
// all frames; LLM streaming runs in its own goroutine so interrupt stops it
// mid-response; per-connection context cancels work on disconnect.
func (h *ChatHandler) Chat(origins []string) fiber.Handler {
	// Origin allowlist (CSWSH defense): when ALLOWED_ORIGINS is set, only
	// listed origins upgrade. Note: the ws library rejects clients WITHOUT an
	// Origin header under an allowlist — non-browser clients (mobile) must
	// send Origin or the list stays empty (dev default "*").
	return fiberws.New(func(conn *fiberws.Conn) {
		claims, _ := conn.Locals("ws_claims").(*application.Claims)
		if claims == nil {
			_ = conn.WriteJSON(ivdomain.ErrorMessage{Type: ivdomain.MsgError, Message: "unauthorized"})
			_ = conn.Close()
			return
		}
		interviewID, ok := ticketInterviewID(claims)
		if !ok {
			_ = conn.WriteJSON(ivdomain.ErrorMessage{Type: ivdomain.MsgError, Message: "invalid ws ticket"})
			_ = conn.Close()
			return
		}
		sessionID := claims.Extra.SessionID
		orgID := claims.OrgID.String()

		// Teardown order is owned by wsWriter.close (D18): it cancels
		// connCtx before touching the writer mutex, so parked senders can
		// never pin close — the handler always exits and Release runs.
		connCtx, cancel := context.WithCancel(context.Background())
		// J4: seed the connection context from the Fiber handshake context
		// (stashed in RequireTicket; fiberws copies UserValues into
		// conn.Locals pre-upgrade) so interview spans nest under the HTTP
		// server span instead of starting orphan roots.
		if fc, ok := conn.Locals("ws_fiber_ctx").(context.Context); ok && fc != nil {
			connCtx, cancel = context.WithCancel(fc)
		}

		ok, err := h.sessions.TryAcquire(connCtx, interviewID.String(), sessionID)
		if err != nil || !ok {
			cancel()
			_ = conn.WriteJSON(ivdomain.ErrorMessage{Type: ivdomain.MsgError, Message: "interview already active on another connection"})
			_ = conn.Close()
			return
		}
		defer func() {
			_ = h.sessions.Release(context.Background(), interviewID.String(), sessionID)
		}()

		w := newWSWriter(conn, connCtx, cancel)
		defer w.close()

		if err := h.svc.StartInterview(connCtx, orgID, interviewID); err != nil {
			w.sendError(err)
			return
		}
		// Pin BOTH connection contexts in one transaction + one download
		// (I12): interviewer prompt and QA grounding must share the same
		// version pin, and duplicate loads doubled connect latency/egress.
		composeCtx, composeSpan := tracer().Start(connCtx, "interview.connect.compose",
			trace.WithAttributes(
				attribute.String("org.id", orgID),
				attribute.String("interview.id", interviewID.String()),
			))
		cc, err := h.svc.ComposeConnectContexts(composeCtx, uuid.MustParse(orgID), interviewID)
		composeSpan.End()
		if err != nil {
			h.log.Error().Err(err).Msg("compose interview connect contexts failed")
			w.sendError(err)
			return
		}
		prompt, qaContext := cc.Prompt, cc.QAContext
		// J11: fail CLOSED — if the QA grounding is unavailable, the candidate
		// question path must refuse rather than run the LLM against partial
		// material. The flag is carried on the session for the handler.
		if cc.GroundingUnavailable {
			h.log.Warn().Str("interview_id", interviewID.String()).Msg("qa grounding unavailable; candidate questions will be refused")
		}
		// History window seeded from the persisted transcript (resume support);
		// appended in-session for the current connection. Guarded by historyMu:
		// the read loop appends while the stream goroutine reads the window.
		history, err := h.svc.RecentContext(connCtx, orgID, interviewID)
		if err != nil {
			history = nil
		}

		s := h.engine.NewSession(session.Config{
			Ctx:                  connCtx,
			OrgID:                orgID,
			InterviewID:          interviewID,
			SessionID:            sessionID,
			Prompt:               prompt,
			QAContext:            qaContext,
			GroundingUnavailable: cc.GroundingUnavailable,
			History:              history,
			Send:                 w.send,
			SendError:            w.sendError,
			CodeRunner:           h.codeRunner,
			AcquireCodeRun: func() bool {
				select {
				case h.runSemaphore <- struct{}{}:
					return true
				default:
					return false
				}
			},
			ReleaseCodeRun: func() {
				<-h.runSemaphore
			},
		})
		defer s.Close()
		s.SendStartAndQuestion()

		// Heartbeat: ping every 30s, drop the socket if the client never pongs.
		pongCh := make(chan struct{}, 1)
		conn.SetPongHandler(func(string) error {
			select {
			case pongCh <- struct{}{}:
			default:
			}
			return nil
		})
		hbDone := make(chan struct{})
		defer close(hbDone)
		go func() {
			for {
				// Keep the session lock alive for the whole connection — a
				// 35-min TTL must not lapse under an active interview.
				if err := h.sessions.Touch(connCtx, interviewID.String(), sessionID); err != nil {
					h.log.Warn().Err(err).Msg("session lock touch failed")
				}
				select {
				case <-time.After(heartbeatInterval):
					if err := conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(5*time.Second)); err != nil {
						return // socket already gone
					}
					select {
					case <-pongCh:
					case <-time.After(pongWait):
						_ = conn.Close() // silent client; read loop errors out
						return
					case <-hbDone:
						return
					}
				case <-hbDone:
					return
				}
			}
		}()

		for {
			_ = conn.SetReadDeadline(time.Now().Add(readDeadline))
			_, raw, err := conn.ReadMessage()
			if err != nil {
				var netErr net.Error
				if errors.As(err, &netErr) && netErr.Timeout() {
					w.send(ivdomain.ErrorMessage{Type: ivdomain.MsgError, Message: "question timeout"})
				}
				return
			}
			// I13: drop oversized frames before parsing — client-controlled
			// payloads must not become unbounded allocations or storage.
			if len(raw) > maxWSFrameBytes {
				w.send(ivdomain.ErrorMessage{Type: ivdomain.MsgError, Message: "frame too large"})
				continue
			}
			msg, err := ivdomain.ParseClientMessage(raw)
			if err != nil {
				w.send(ivdomain.ErrorMessage{Type: ivdomain.MsgError, Message: "invalid message"})
				continue
			}
			switch m := msg.(type) {
			case ivdomain.PingMessage:
				w.send(map[string]string{"type": ivdomain.MsgPong})
			case ivdomain.ResumeMessage:
				if m.SessionID != sessionID {
					w.send(ivdomain.ErrorMessage{Type: ivdomain.MsgError, Message: "session mismatch"})
					continue
				}
				s.HandleResume()
			case ivdomain.TelemetryMessage:
				s.HandleTelemetry(m)
			case ivdomain.CodeChangeMessage:
				s.DebouncedTouch()
			case ivdomain.CodeRunMessage:
				s.HandleCodeRun(m)
			case ivdomain.AnswerMessage:
				if !s.HandleAnswer(m) {
					return
				}
			case ivdomain.CandidateQuestionMessage:
				s.HandleCandidateQuestion(m)
			case ivdomain.InterruptMessage:
				s.HandleInterrupt()
			}
		}
	}, fiberws.Config{Origins: origins})
}

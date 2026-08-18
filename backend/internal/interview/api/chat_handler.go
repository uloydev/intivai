package api

import (
	"context"
	"errors"
	"net"
	"strings"
	"sync"
	"time"

	"encoding/json"
	fiberws "github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	evalllm "github.com/intivai/backend/internal/evaluation/infrastructure/llm"
	"github.com/intivai/backend/internal/iam/api"
	"github.com/intivai/backend/internal/iam/application"
	ivapp "github.com/intivai/backend/internal/interview/application"
	ivdomain "github.com/intivai/backend/internal/interview/domain"
	gensvc "github.com/intivai/backend/internal/interview/domain/service"
	"github.com/intivai/backend/internal/llm"
	sbapp "github.com/intivai/backend/internal/sandbox/application"
	sbdomain "github.com/intivai/backend/internal/sandbox/domain"
	sharederrors "github.com/intivai/backend/internal/shared/errors"
	"github.com/intivai/backend/internal/shared/httpapi"
	"github.com/rs/zerolog"
)

// Read deadline per candidate frame (Research §2: PerQuestionTimeout = 3 min).
// Stricter than the domain idle rule — a silent candidate disconnects at 3
// minutes; the 5-minute idle/expiry path still guards server-side state.
// errorFrame maps a domain error to a WS error frame with its machine
// readable code (FE can distinguish CONSENT_REQUIRED from INTERVIEW_EXPIRED).
func errorFrame(err error) ivdomain.ErrorMessage {
	frame := ivdomain.ErrorMessage{Type: ivdomain.MsgError, Message: err.Error()}
	var de *sharederrors.DomainError
	if errors.As(err, &de) {
		frame.Code = de.Code
	}
	return frame
}

const readDeadline = ivdomain.PerQuestionTimeout

// Server heartbeat (Research §2): ping every 30s; drop the socket when the
// client does not answer with a pong within 10s.
const (
	heartbeatInterval = 30 * time.Second
	pongWait          = 10 * time.Second
)

// turnState serializes the "next question" dispatch between the streaming
// goroutine and the interrupt path — exactly one sends it. onSent fires with
// the dispatched question (used to track the last question for history pairs).
type turnState struct {
	mu           sync.Mutex
	next         *ivdomain.Question
	questionSent bool
	remainingSec int
	onSent       func(q *ivdomain.Question)
}

func (t *turnState) sendQuestionOnce(send func(any)) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.questionSent && t.next != nil {
		t.questionSent = true
		arch, limit := ivdomain.DetermineQuestionArchetype(*t.next)
		send(ivdomain.QuestionMessage{
			Type:                ivdomain.MsgQuestion,
			Content:             t.next.Content,
			Idx:                 t.next.Idx,
			Archetype:           arch,
			TimeLimitSec:        limit,
			SessionRemainingSec: t.remainingSec,
		})
		if t.onSent != nil {
			t.onSent(t.next)
		}
	}
}

type ChatHandler struct {
	svc        *ivapp.InterviewService
	llm        llm.Provider
	tokens     application.TokenProvider
	log        zerolog.Logger
	sessions   SessionRegistry
	codeRunner sbapp.CodeRunner
}

func NewChatHandler(svc *ivapp.InterviewService, llmClient llm.Provider, tokens application.TokenProvider, log zerolog.Logger, sessions ...SessionRegistry) *ChatHandler {
	var reg SessionRegistry = NewMemorySessionRegistry()
	if len(sessions) > 0 && sessions[0] != nil {
		reg = sessions[0]
	}
	return &ChatHandler{svc: svc, llm: llmClient, tokens: tokens, log: log, sessions: reg}
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
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	appID, err := uuid.Parse(req.ApplicationID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid application_id"})
	}
	result, err := h.svc.CreateInterview(c.UserContext(), actor, ivapp.CreateInterviewCommand{
		ApplicationID: appID, QuestionCount: req.QuestionCount,
	})
	if err != nil {
		return httpapi.Error(c, err)
	}
	return httpapi.Created(c, result)
}

// Consent — POST /candidate/interviews/:id/consent (candidate, invitation
// token). Records GDPR consent; the chat refuses to start without it.
func (h *ChatHandler) Consent(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid interview id"})
	}
	var req struct {
		InvitationToken string `json:"invitation_token"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	if err := h.svc.GiveConsent(c.UserContext(), id, strings.TrimSpace(req.InvitationToken)); err != nil {
		return httpapi.Error(c, err)
	}
	return c.Status(200).JSON(fiber.Map{"data": map[string]bool{"consent_given": true}})
}

// Ticket — POST /candidate/interviews/:id/ticket (candidate, invitation token).
func (h *ChatHandler) Ticket(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid interview id"})
	}
	var req struct {
		InvitationToken string `json:"invitation_token"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	result, err := h.svc.IssueTicket(c.UserContext(), ivapp.IssueTicketCommand{
		InterviewID: id, InvitationToken: strings.TrimSpace(req.InvitationToken),
	})
	if err != nil {
		return httpapi.Error(c, err)
	}
	return httpapi.OK(c, result)
}

// Telemetry — POST /candidate/interviews/:id/telemetry (candidate beacon/HTTP fallback).
func (h *ChatHandler) Telemetry(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid interview id"})
	}
	var req struct {
		InvitationToken string                 `json:"invitation_token"`
		Ticket          string                 `json:"ticket"`
		EventType       string                 `json:"event_type"`
		Timestamp       string                 `json:"timestamp"`
		QuestionIdx     int                    `json:"question_idx"`
		Details         map[string]interface{} `json:"details"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
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
		return c.Status(401).JSON(fiber.Map{"error": "missing authentication token or ticket"})
	}

	var evTime time.Time
	if req.Timestamp != "" {
		evTime, _ = time.Parse(time.RFC3339, req.Timestamp)
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
	return c.Status(200).JSON(fiber.Map{"status": "ok"})
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
		return c.Status(401).JSON(fiber.Map{"error": "missing ws ticket"})
	}
	claims, err := h.tokens.Parse(token)
	if err != nil || claims.Type != application.TokenTypeWSTicket {
		return c.Status(401).JSON(fiber.Map{"error": "invalid ws ticket"})
	}
	if claims.Extra["interview_id"] != c.Params("id") {
		return c.Status(401).JSON(fiber.Map{"error": "ticket not bound to this interview"})
	}
	c.Locals("ws_claims", claims)
	return c.Next()
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
		interviewID := uuid.MustParse(claims.Extra["interview_id"].(string))
		sessionID, _ := claims.Extra["session_id"].(string)
		orgID := claims.OrgID.String()

		connCtx, cancel := context.WithCancel(context.Background())
		defer cancel()

		connID := uuid.NewString()
		ok, err := h.sessions.TryAcquire(connCtx, interviewID.String(), connID)
		if err != nil || !ok {
			_ = conn.WriteJSON(ivdomain.ErrorMessage{Type: ivdomain.MsgError, Message: "interview already active on another connection"})
			_ = conn.Close()
			return
		}
		defer func() {
			_ = h.sessions.Release(context.Background(), interviewID.String(), connID)
		}()

		// Single writer: all frames go through this goroutine.
		writeCh := make(chan any, 64)
		writerDone := make(chan struct{})
		var writeMu sync.Mutex
		var writeClosed bool

		go func() {
			defer close(writerDone)
			for frame := range writeCh {
				if err := conn.WriteJSON(frame); err != nil {
					return
				}
			}
		}()
		defer func() {
			writeMu.Lock()
			writeClosed = true
			close(writeCh)
			writeMu.Unlock()
			<-writerDone
		}()
		send := func(frame any) {
			writeMu.Lock()
			defer writeMu.Unlock()
			if writeClosed || connCtx.Err() != nil {
				return
			}
			select {
			case writeCh <- frame:
			case <-connCtx.Done():
			}
		}

		if err := h.svc.StartInterview(connCtx, orgID, interviewID); err != nil {
			send(errorFrame(err))
			return
		}
		// Compose the prompt ONCE per connection (version pinned at connect).
		prompt := h.composePromptOnce(connCtx, orgID)
		// History window seeded from the persisted transcript (resume support);
		// appended in-session for the current connection. Guarded by historyMu:
		// the read loop appends while the stream goroutine reads the window.
		var historyMu sync.Mutex
		history, err := h.svc.RecentContext(connCtx, orgID, interviewID)
		if err != nil {
			history = nil
		}
		var lastQuestion *ivdomain.Question
		var currentArchetype string
		questionSent := func(q *ivdomain.Question) {
			historyMu.Lock()
			lastQuestion = q
			arch, _ := ivdomain.DetermineQuestionArchetype(*q)
			currentArchetype = arch
			historyMu.Unlock()
		}
		h.sendStartAndQuestion(connCtx, send, sessionID, orgID, interviewID, questionSent)

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
				if err := h.sessions.Touch(connCtx, interviewID.String(), connID); err != nil {
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

		var streamCancel context.CancelFunc
		var turn *turnState

		for {
			_ = conn.SetReadDeadline(time.Now().Add(readDeadline))
			_, raw, err := conn.ReadMessage()
			if err != nil {
				var netErr net.Error
				if errors.As(err, &netErr) && netErr.Timeout() {
					send(ivdomain.ErrorMessage{Type: ivdomain.MsgError, Message: "question timeout"})
				}
				return
			}
			msg, err := ivdomain.ParseClientMessage(raw)
			if err != nil {
				send(ivdomain.ErrorMessage{Type: ivdomain.MsgError, Message: "invalid message"})
				continue
			}
			switch m := msg.(type) {
			case ivdomain.PingMessage:
				send(map[string]string{"type": ivdomain.MsgPong})
			case ivdomain.ResumeMessage:
				if m.SessionID != sessionID {
					send(ivdomain.ErrorMessage{Type: ivdomain.MsgError, Message: "session mismatch"})
					continue
				}
				h.sendStartAndQuestion(connCtx, send, sessionID, orgID, interviewID, questionSent)
			case ivdomain.InterruptMessage:
				if streamCancel != nil {
					streamCancel() // stops the LLM stream mid-response
					streamCancel = nil
				}
				send(ivdomain.ResponseMessage{Type: ivdomain.MsgResponse, Content: "Interrupted."})
				// The stream goroutine suppresses the next question when its
				// ctx is canceled; send it here exactly once instead. If the
				// stream already completed normally, questionSent guards the
				// double-send.
				if turn != nil {
					turn.sendQuestionOnce(send)
				}
				turn = nil
			case ivdomain.TelemetryMessage:
				var evTime time.Time
				if m.Timestamp != "" {
					evTime, _ = time.Parse(time.RFC3339, m.Timestamp)
				}
				if evTime.IsZero() {
					evTime = time.Now()
				}
				event := ivdomain.ProctoringEvent{
					Type:        ivdomain.ProctoringEventType(m.EventType),
					Timestamp:   evTime,
					QuestionIdx: m.QuestionIdx,
					Details:     m.Details,
				}
				_ = h.svc.RecordTelemetry(connCtx, orgID, interviewID, event)
			case ivdomain.CodeChangeMessage:
				h.svc.TouchInterview(connCtx, orgID, interviewID)
			case ivdomain.CodeRunMessage:
				// Gate: code execution is only allowed while a coding
				// question is active (design decision).
				historyMu.Lock()
				arch := currentArchetype
				historyMu.Unlock()
				if arch != ivdomain.ArchetypeCoding {
					send(ivdomain.CodeResultMessage{
						Type:  ivdomain.MsgCodeResult,
						Error: "code execution is only allowed on coding challenges",
					})
					continue
				}
				// Execute off the read loop: a slow run must not block
				// heartbeat/interrupt frames or trip the read deadline.
				// Cap test cases + total wall time (each case already has a
				// per-case timeout; N cases × timeout must stay bounded).
				go h.runCode(connCtx, send, orgID, interviewID, m)
			case ivdomain.AnswerMessage:
				next, err := h.svc.AnswerAndAdvanceWithPacing(connCtx, orgID, interviewID, m.Content, m.PacingTelemetry)
				if err != nil {
					send(errorFrame(err))
					return
				}
				historyMu.Lock()
				if lastQuestion != nil {
					history = append(history,
						gensvc.ContextMessage{Role: gensvc.RoleAssistant, Content: lastQuestion.Content},
						gensvc.ContextMessage{Role: gensvc.RoleUser, Content: m.Content},
					)
				}
				historyMu.Unlock()
				remSec := h.svc.SessionRemaining(connCtx, orgID, interviewID)
				turn = &turnState{next: next, remainingSec: remSec, onSent: questionSent}
				streamCtx, cancelStream := context.WithCancel(connCtx)
				streamCancel = cancelStream
				go h.streamAndRespond(streamCtx, send, prompt, m.Content, next, turn, &history, &historyMu, orgID, interviewID)
			}
		}
	}, fiberws.Config{Origins: origins})
}

// maxSandboxTestCases bounds per-run subprocess spawns (DoS guard).
const maxSandboxTestCases = 20

// codeRunTimeout caps the whole run (test cases included) so many slow cases
// cannot pin the connection goroutines.
const codeRunTimeout = 30 * time.Second

// runCode executes a code.run frame off the read loop and streams the result
// frame back (same single-writer send path). Fail-closed guards: no sidecar
// wired, or the current question is not a coding archetype.
func (h *ChatHandler) runCode(ctx context.Context, send func(any), orgID string, interviewID uuid.UUID, m ivdomain.CodeRunMessage) {
	if h.codeRunner == nil {
		send(ivdomain.CodeResultMessage{Type: ivdomain.MsgCodeResult, Error: "code execution is unavailable"})
		return
	}
	tcs := make([]sbdomain.TestCase, 0, len(m.TestCases))
	for i, tc := range m.TestCases {
		if i >= maxSandboxTestCases {
			break
		}
		tcs = append(tcs, sbdomain.TestCase{
			ID:             tc.ID,
			Input:          tc.Input,
			ExpectedOutput: tc.ExpectedOutput,
			Hidden:         tc.Hidden,
		})
	}
	execReq := sbdomain.ExecutionRequest{
		Language:  sbdomain.Language(m.Language),
		Code:      m.Code,
		Stdin:     m.Stdin,
		TestCases: tcs,
	}
	runCtx, cancel := context.WithTimeout(ctx, codeRunTimeout)
	defer cancel()
	res, err := h.codeRunner.Execute(runCtx, execReq)
	if err != nil {
		send(ivdomain.CodeResultMessage{
			Type:  ivdomain.MsgCodeResult,
			Error: err.Error(),
		})
		return
	}
	var rawTests []interface{}
	for _, tr := range res.TestResults {
		rawTests = append(rawTests, tr)
	}
	send(ivdomain.CodeResultMessage{
		Type:        ivdomain.MsgCodeResult,
		Stdout:      res.Stdout,
		Stderr:      res.Stderr,
		ExitCode:    res.ExitCode,
		DurationMs:  res.DurationMs,
		AllPassed:   res.AllPassed,
		TestResults: rawTests,
		Error:       res.Error,
	})
	_ = h.svc.RecordCodingSession(ctx, orgID, interviewID, ivdomain.CodingSession{
		QuestionIdx: m.QuestionIdx,
		Language:    m.Language,
		Code:        m.Code,
	})
}

// composePromptOnce builds the system prompt at connect time; failures fall
// back to the default + safety rails.
func (h *ChatHandler) composePromptOnce(ctx context.Context, orgID string) string {
	prompt, err := h.svc.ComposePrompt(ctx, uuid.MustParse(orgID))
	if err != nil {
		h.log.Error().Err(err).Msg("compose prompt failed, using default")
		return gensvc.ComposeSystemPrompt(gensvc.ComposerInput{})
	}
	return prompt
}

func (h *ChatHandler) sendStartAndQuestion(ctx context.Context, send func(any), sessionID, orgID string, interviewID uuid.UUID, onSent func(*ivdomain.Question)) {
	next, total, status, err := h.svc.CurrentState(ctx, orgID, interviewID)
	if err != nil {
		return
	}
	remSec := h.svc.SessionRemaining(ctx, orgID, interviewID)
	send(ivdomain.InterviewStartMessage{
		Type:             ivdomain.MsgStart,
		SessionID:        sessionID,
		TotalQuestions:   total,
		SessionBudgetSec: int(ivdomain.MaxInterviewDuration.Seconds()),
	})
	if status == ivdomain.StatusInProgress && next != nil {
		arch, limit := ivdomain.DetermineQuestionArchetype(*next)
		send(ivdomain.QuestionMessage{
			Type:                ivdomain.MsgQuestion,
			Content:             next.Content,
			Idx:                 next.Idx,
			Archetype:           arch,
			TimeLimitSec:        limit,
			SessionRemainingSec: remSec,
		})
		if onSent != nil {
			onSent(next)
		}
	}
}

// streamAndRespond runs the LLM stream in its own goroutine. On normal
// completion it sends response + next question; a canceled ctx (interrupt /
// disconnect) suppresses both — the interrupt path dispatches the question.
// History is trimmed to the sliding window (last 10 Q&A); the total message
// budget (8K tokens) is enforced before streaming — overruns degrade to an
// error frame and the interview keeps moving.
func (h *ChatHandler) streamAndRespond(ctx context.Context, send func(any), prompt, answer string, next *ivdomain.Question, turn *turnState, history *[]gensvc.ContextMessage, historyMu *sync.Mutex, orgID string, interviewID uuid.UUID) {
	msgs := []gensvc.ContextMessage{{Role: gensvc.RoleSystem, Content: prompt}}
	historyMu.Lock()
	historySnapshot := gensvc.TrimContext(*history, gensvc.DefaultContextWindow)
	historyMu.Unlock()
	msgs = append(msgs, historySnapshot...)

	chatMsgs := make([]llm.Message, 0, len(msgs))
	for _, m := range msgs {
		chatMsgs = append(chatMsgs, llm.Message{Role: m.Role, Content: m.Content})
	}
	if gensvc.ExceedsBudget(msgs, gensvc.DefaultTokenBudget, h.llm.CountTokens) {
		h.log.Warn().Int("messages", len(msgs)).Msg("context budget exceeded")
		send(ivdomain.ErrorMessage{Type: ivdomain.MsgError, Message: "context budget exceeded"})
		turn.sendQuestionOnce(send)
		return
	}

	ch, err := h.llm.ChatStream(ctx, llm.ChatRequest{Messages: chatMsgs})
	if err != nil {
		h.log.Error().Err(err).Msg("chat stream failed")
		send(ivdomain.ErrorMessage{Type: ivdomain.MsgError, Message: "llm unavailable"})
		// The answer was recorded — keep the interview moving by dispatching
		// the next question (same recovery as interrupt).
		turn.sendQuestionOnce(send)
		return
	}
	var final strings.Builder
	for token := range ch {
		final.WriteString(token)
		send(ivdomain.TokenMessage{Type: ivdomain.MsgToken, Content: token})
	}
	if ctx.Err() != nil {
		return // interrupted: response/next question suppressed
	}
	send(ivdomain.ResponseMessage{Type: ivdomain.MsgResponse, Content: final.String()})
	turn.sendQuestionOnce(send)
	if next == nil {
		h.sendEvaluation(ctx, send, orgID, interviewID)
	}
}

// sendEvaluation computes the post-interview report inline (≤20s) and sends
// the evaluation frame with real scores. On LLM failure: pending frame +
// async retry via the evaluation worker.
func (h *ChatHandler) sendEvaluation(ctx context.Context, send func(any), orgID string, interviewID uuid.UUID) {
	evalCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	pairs, err := h.svc.Transcript(evalCtx, orgID, interviewID)
	if err != nil {
		h.pendingEvaluation(send, orgID, interviewID)
		return
	}
	report, err := evalllm.NewEvaluator(h.llm).Evaluate(evalCtx, pairs)
	if err != nil {
		h.log.Warn().Err(err).Msg("inline evaluation failed, deferring to worker")
		h.pendingEvaluation(send, orgID, interviewID)
		return
	}
	raw, err := json.Marshal(report)
	if err != nil {
		h.pendingEvaluation(send, orgID, interviewID)
		return
	}
	if err := h.svc.EvaluateAndPersist(evalCtx, orgID, interviewID, raw); err != nil {
		h.log.Warn().Err(err).Msg("persist evaluation")
	}
	scores := make(map[string]float64, len(report.Dimensions))
	for name, d := range report.Dimensions {
		scores[name] = d.Score
	}
	send(ivdomain.EvaluationMessage{
		Type:           ivdomain.MsgEvaluation,
		Scores:         scores,
		Overall:        report.OverallScore,
		Recommendation: report.Recommendation,
		Status:         ivdomain.EvalComplete,
	})
}

func (h *ChatHandler) pendingEvaluation(send func(any), orgID string, interviewID uuid.UUID) {
	if err := h.svc.EnqueueEvaluation(context.Background(), orgID, interviewID); err != nil {
		h.log.Warn().Err(err).Msg("enqueue evaluation retry")
	}
	send(ivdomain.EvaluationMessage{
		Type:   ivdomain.MsgEvaluation,
		Scores: map[string]float64{},
		Status: ivdomain.EvalPending,
	})
}

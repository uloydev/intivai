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
	"github.com/intivai/backend/internal/iam/api"
	"github.com/intivai/backend/internal/iam/application"
	ivapp "github.com/intivai/backend/internal/interview/application"
	ivdomain "github.com/intivai/backend/internal/interview/domain"
	gensvc "github.com/intivai/backend/internal/interview/domain/service"
	"github.com/intivai/backend/internal/llm"
	"github.com/intivai/backend/internal/shared/httpapi"
	"github.com/rs/zerolog"
)

// Read deadline per candidate frame (Research §2: PerQuestionTimeout = 3 min).
// Stricter than the domain idle rule — a silent candidate disconnects at 3
// minutes; the 5-minute idle/expiry path still guards server-side state.
const readDeadline = ivdomain.PerQuestionTimeout

// turnState serializes the "next question" dispatch between the streaming
// goroutine and the interrupt path — exactly one sends it. onSent fires with
// the dispatched question (used to track the last question for history pairs).
type turnState struct {
	mu           sync.Mutex
	next         *ivdomain.Question
	questionSent bool
	onSent       func(q *ivdomain.Question)
}

func (t *turnState) sendQuestionOnce(send func(any)) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.questionSent && t.next != nil {
		t.questionSent = true
		send(ivdomain.QuestionMessage{Type: ivdomain.MsgQuestion, Content: t.next.Content, Idx: t.next.Idx})
		if t.onSent != nil {
			t.onSent(t.next)
		}
	}
}

type ChatHandler struct {
	svc      *ivapp.InterviewService
	llm      llm.Provider
	tokens   application.TokenProvider
	log      zerolog.Logger
	sessions *sessionRegistry
}

func NewChatHandler(svc *ivapp.InterviewService, llmClient llm.Provider, tokens application.TokenProvider, log zerolog.Logger) *ChatHandler {
	return &ChatHandler{svc: svc, llm: llmClient, tokens: tokens, log: log, sessions: newSessionRegistry()}
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

// RequireTicket — pre-upgrade gate: Bearer must be a ws_ticket bound to this
// interview. Non-upgrade requests get 401; upgrades proceed to Chat.
func (h *ChatHandler) RequireTicket(c *fiber.Ctx) error {
	header := c.Get("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		return c.Status(401).JSON(fiber.Map{"error": "missing ws ticket"})
	}
	claims, err := h.tokens.Parse(strings.TrimPrefix(header, "Bearer "))
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

		if !h.sessions.TryAcquire(interviewID.String()) {
			_ = conn.WriteJSON(ivdomain.ErrorMessage{Type: ivdomain.MsgError, Message: "interview already active on another connection"})
			_ = conn.Close()
			return
		}
		defer h.sessions.Release(interviewID.String())

		connCtx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Single writer: all frames go through this goroutine.
		writeCh := make(chan any, 64)
		writerDone := make(chan struct{})
		go func() {
			defer close(writerDone)
			for frame := range writeCh {
				if err := conn.WriteJSON(frame); err != nil {
					return
				}
			}
		}()
		defer func() {
			close(writeCh)
			<-writerDone
		}()
		send := func(frame any) {
			select {
			case writeCh <- frame:
			case <-connCtx.Done():
			}
		}

		if err := h.svc.StartInterview(connCtx, orgID, interviewID); err != nil {
			send(ivdomain.ErrorMessage{Type: ivdomain.MsgError, Message: err.Error()})
			return
		}
		// Compose the prompt ONCE per connection (version pinned at connect).
		prompt := h.composePromptOnce(connCtx, orgID)
		// History window seeded from the persisted transcript (resume support);
		// appended in-session for the current connection.
		history, err := h.svc.RecentContext(connCtx, orgID, interviewID)
		if err != nil {
			history = nil
		}
		var lastQuestion *ivdomain.Question
		questionSent := func(q *ivdomain.Question) { lastQuestion = q }
		h.sendStartAndQuestion(connCtx, send, sessionID, orgID, interviewID, questionSent)

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
			case ivdomain.AnswerMessage:
				next, err := h.svc.AnswerAndAdvance(connCtx, orgID, interviewID, m.Content)
				if err != nil {
					send(ivdomain.ErrorMessage{Type: ivdomain.MsgError, Message: err.Error()})
					return
				}
				if lastQuestion != nil {
					history = append(history,
						gensvc.ContextMessage{Role: gensvc.RoleAssistant, Content: lastQuestion.Content},
						gensvc.ContextMessage{Role: gensvc.RoleUser, Content: m.Content},
					)
				}
				turn = &turnState{next: next, onSent: questionSent}
				streamCtx, cancelStream := context.WithCancel(connCtx)
				streamCancel = cancelStream
				go h.streamAndRespond(streamCtx, send, prompt, m.Content, next, turn, &history)
			}
		}
	}, fiberws.Config{Origins: origins})
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
	send(ivdomain.InterviewStartMessage{Type: ivdomain.MsgStart, SessionID: sessionID, TotalQuestions: total})
	if status == ivdomain.StatusInProgress && next != nil {
		send(ivdomain.QuestionMessage{Type: ivdomain.MsgQuestion, Content: next.Content, Idx: next.Idx})
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
func (h *ChatHandler) streamAndRespond(ctx context.Context, send func(any), prompt, answer string, next *ivdomain.Question, turn *turnState, history *[]gensvc.ContextMessage) {
	msgs := []gensvc.ContextMessage{{Role: gensvc.RoleSystem, Content: prompt}}
	msgs = append(msgs, gensvc.TrimContext(*history, gensvc.DefaultContextWindow)...)

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
		send(ivdomain.EvaluationMessage{Type: ivdomain.MsgEvaluation, Scores: map[string]float64{}})
	}
}

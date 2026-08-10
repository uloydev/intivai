package api

import (
	"context"
	"errors"
	"net"
	"strings"
	"time"

	fiberws "github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/intivai/backend/internal/iam/api"
	"github.com/intivai/backend/internal/iam/application"
	ivapp "github.com/intivai/backend/internal/interview/application"
	ivdomain "github.com/intivai/backend/internal/interview/domain"
	"github.com/intivai/backend/internal/llm"
	"github.com/intivai/backend/internal/shared/httpapi"
	"github.com/rs/zerolog"
)

const idleTimeout = 5 * time.Minute

type ChatHandler struct {
	svc    *ivapp.InterviewService
	llm    llm.Provider
	tokens application.TokenProvider
	log    zerolog.Logger
}

func NewChatHandler(svc *ivapp.InterviewService, llmClient llm.Provider, tokens application.TokenProvider, log zerolog.Logger) *ChatHandler {
	return &ChatHandler{svc: svc, llm: llmClient, tokens: tokens, log: log}
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

// Ticket — POST /interviews/:id/ticket (candidate, invitation token).
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

// Chat — WS /interviews/:id/chat. Protocol: ping→pong, answer → streamed LLM
// tokens → response → next question; interrupt stops streaming; resume
// re-sends start + current question; idle timeout closes.
func (h *ChatHandler) Chat() fiber.Handler {
	return fiberws.New(func(conn *fiberws.Conn) {
		claims, _ := conn.Locals("ws_claims").(*application.Claims)
		if claims == nil {
			_ = conn.WriteJSON(ivdomain.ErrorMessage{Type: ivdomain.MsgError, Message: "unauthorized"})
			_ = conn.Close()
			return
		}
		interviewID := uuid.MustParse(claims.Extra["interview_id"].(string))
		orgID := claims.OrgID.String()

		if err := h.svc.StartInterview(context.Background(), orgID, interviewID); err != nil {
			_ = conn.WriteJSON(ivdomain.ErrorMessage{Type: ivdomain.MsgError, Message: err.Error()})
			_ = conn.Close()
			return
		}
		if err := h.sendStartAndQuestion(conn, orgID, interviewID); err != nil {
			return
		}

		for {
			_ = conn.SetReadDeadline(time.Now().Add(idleTimeout))
			_, raw, err := conn.ReadMessage()
			if err != nil {
				var netErr net.Error
				if errors.As(err, &netErr) && netErr.Timeout() {
					_ = conn.WriteJSON(ivdomain.ErrorMessage{Type: ivdomain.MsgError, Message: "idle timeout"})
				}
				return
			}
			msg, err := ivdomain.ParseClientMessage(raw)
			if err != nil {
				_ = conn.WriteJSON(ivdomain.ErrorMessage{Type: ivdomain.MsgError, Message: "invalid message"})
				continue
			}
			switch m := msg.(type) {
			case ivdomain.PingMessage:
				_ = conn.WriteJSON(map[string]string{"type": ivdomain.MsgPong})
			case ivdomain.ResumeMessage:
				_ = h.sendStartAndQuestion(conn, orgID, interviewID)
			case ivdomain.InterruptMessage:
				_ = conn.WriteJSON(ivdomain.ResponseMessage{Type: ivdomain.MsgResponse, Content: "Interrupted."})
			case ivdomain.AnswerMessage:
				if err := h.handleAnswer(conn, orgID, interviewID, m.Content); err != nil {
					_ = conn.WriteJSON(ivdomain.ErrorMessage{Type: ivdomain.MsgError, Message: err.Error()})
					return
				}
			}
		}
	})
}

func (h *ChatHandler) sendStartAndQuestion(conn *fiberws.Conn, orgID string, interviewID uuid.UUID) error {
	next, total, status, err := h.svc.CurrentState(context.Background(), orgID, interviewID)
	if err != nil {
		return err
	}
	sid := ""
	if claims, ok := conn.Locals("ws_claims").(*application.Claims); ok {
		sid, _ = claims.Extra["session_id"].(string)
	}
	_ = conn.WriteJSON(ivdomain.InterviewStartMessage{Type: ivdomain.MsgStart, SessionID: sid, TotalQuestions: total})
	if status == ivdomain.StatusInProgress && next != nil {
		_ = conn.WriteJSON(ivdomain.QuestionMessage{Type: ivdomain.MsgQuestion, Content: next.Content, Idx: next.Idx})
	}
	return nil
}

func (h *ChatHandler) handleAnswer(conn *fiberws.Conn, orgID string, interviewID uuid.UUID, content string) error {
	next, err := h.svc.AnswerAndAdvance(context.Background(), orgID, interviewID, content)
	if err != nil {
		return err
	}
	systemPrompt, err := h.svc.ComposePrompt(context.Background(), uuid.MustParse(orgID))
	if err != nil {
		systemPrompt = ""
	}
	ch, err := h.llm.ChatStream(context.Background(), llm.ChatRequest{
		Messages: []llm.Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: content},
		},
	})
	if err != nil {
		_ = conn.WriteJSON(ivdomain.ErrorMessage{Type: ivdomain.MsgError, Message: "llm unavailable: " + err.Error()})
		return nil
	}
	var final strings.Builder
	for token := range ch {
		final.WriteString(token)
		_ = conn.WriteJSON(ivdomain.TokenMessage{Type: ivdomain.MsgToken, Content: token})
	}
	_ = conn.WriteJSON(ivdomain.ResponseMessage{Type: ivdomain.MsgResponse, Content: final.String()})
	if next != nil {
		_ = conn.WriteJSON(ivdomain.QuestionMessage{Type: ivdomain.MsgQuestion, Content: next.Content, Idx: next.Idx})
	} else {
		_ = conn.WriteJSON(ivdomain.EvaluationMessage{Type: ivdomain.MsgEvaluation, Scores: map[string]float64{}})
	}
	return nil
}

package session

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	evalllm "github.com/intivai/backend/internal/evaluation/infrastructure/llm"
	ivapp "github.com/intivai/backend/internal/interview/application"
	ivdomain "github.com/intivai/backend/internal/interview/domain"
	gensvc "github.com/intivai/backend/internal/interview/domain/service"
	"github.com/intivai/backend/internal/llm"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Service represents the domain interview operations required by DialogueEngine and InterviewSession.
type Service interface {
	ProcessTopicDialogue(ctx context.Context, orgID string, interviewID uuid.UUID, content string, action string, pacing *ivdomain.PacingMetrics) (*ivapp.TopicDialogueResult, error)
	SessionRemaining(ctx context.Context, orgID string, interviewID uuid.UUID) int
	TouchInterview(ctx context.Context, orgID string, interviewID uuid.UUID) error
	RecordTelemetry(ctx context.Context, orgID string, interviewID uuid.UUID, event ivdomain.ProctoringEvent) error
	RecordCandidateQA(ctx context.Context, orgID string, interviewID uuid.UUID, question, answer string) error
	CandidateQARemaining(ctx context.Context, orgID string, interviewID uuid.UUID) (int, error)
	CurrentState(ctx context.Context, orgID string, interviewID uuid.UUID) (*ivdomain.Question, int, ivdomain.Status, error)
	Transcript(ctx context.Context, orgID string, interviewID uuid.UUID) ([]ivdomain.TranscriptPair, error)
	EvaluateAndPersist(ctx context.Context, orgID string, interviewID uuid.UUID, reportJSON []byte) error
	EnqueueEvaluation(ctx context.Context, orgID string, interviewID uuid.UUID) error
	RecordCodingSession(ctx context.Context, orgID string, interviewID uuid.UUID, sess ivdomain.CodingSession) error
}

// DialogueEngine manages LLM prompt composition, token streaming, and post-interview evaluations.
type DialogueEngine struct {
	svc Service
	llm llm.Provider
	log zerolog.Logger
}

// NewDialogueEngine creates a new dialogue engine instance.
func NewDialogueEngine(svc Service, llmProvider llm.Provider, log zerolog.Logger) *DialogueEngine {
	return &DialogueEngine{
		svc: svc,
		llm: llmProvider,
		log: log,
	}
}

// StreamAndRespond runs LLM streaming in a managed goroutine.
func (e *DialogueEngine) StreamAndRespond(ctx context.Context, s *InterviewSession, answer string, next *ivdomain.Question, turn *TurnState) {
	ctx, span := tracer().Start(ctx, "interview.turn.stream",
		trace.WithAttributes(
			attribute.String("org.id", s.orgID),
			attribute.String("interview.id", s.interviewID.String()),
			attribute.Int("answer.runes", len([]rune(answer))),
		))
	defer func() {
		if r := recover(); r != nil {
			span.SetStatus(codes.Error, "panic during stream")
			span.End()
			panic(r)
		}
		span.End()
	}()

	msgs := []gensvc.ContextMessage{{Role: gensvc.RoleSystem, Content: s.prompt}}
	s.historyMu.Lock()
	historySnapshot := gensvc.TrimContext(s.history, gensvc.DefaultContextWindow)
	lastQ := s.lastQuestion
	s.historyMu.Unlock()
	msgs = append(msgs, historySnapshot...)

	switch {
	case !turn.IsTopicComplete:
		topicPrompt := "the current question"
		if lastQ != nil {
			topicPrompt = fmt.Sprintf("Question %d: \"%s\"", lastQ.Idx, lastQ.Content)
		}
		msgs = append(msgs, gensvc.ContextMessage{
			Role: gensvc.RoleSystem,
			Content: fmt.Sprintf("You are an expert AI technical interviewer currently exploring %s with the candidate (Turn %d of %d).\n"+
				"The candidate just replied: \"%s\".\n\n"+
				"Instructions:\n"+
				"1. If the candidate asked a clarifying question (e.g. scoping, expected scale, assumptions), answer it directly, clearly, and concisely without advancing or changing the topic.\n"+
				"2. If the candidate gave a technical solution, acknowledge their specific points and probe deeper into trade-offs, race conditions, edge cases, failure modes, or bottlenecks on THIS SAME TOPIC.\n"+
				"3. Keep your response focused and conversational (2-4 sentences). Do NOT transition to any other question.",
				topicPrompt, turn.TopicTurn, turn.MaxTopicTurns, answer),
		})
	case next != nil:
		msgs = append(msgs, gensvc.ContextMessage{
			Role:    gensvc.RoleSystem,
			Content: fmt.Sprintf("The discussion on the previous question is now finalized. Briefly acknowledge the candidate's final response and provide a natural, 1-sentence transition to the next topic: \"%s\". Do NOT ask the next question yourself.", next.Content),
		})
	default:
		msgs = append(msgs, gensvc.ContextMessage{
			Role:    gensvc.RoleSystem,
			Content: "The entire technical interview is now complete. Thank the candidate warmly for their time and conclude the session. Do NOT ask any further questions.",
		})
	}

	chatMsgs := make([]llm.Message, 0, len(msgs))
	for _, m := range msgs {
		chatMsgs = append(chatMsgs, llm.Message{Role: m.Role, Content: m.Content})
	}
	var countFn func(string) int
	if e.llm != nil {
		countFn = e.llm.CountTokens
	} else {
		countFn = func(string) int { return 0 }
	}
	if gensvc.ExceedsBudget(msgs, gensvc.DefaultTokenBudget, countFn) {
		e.log.Warn().Str("interview_id", s.interviewID.String()).Str("org_id", s.orgID).Int("messages", len(msgs)).Msg("context budget exceeded")
		s.send(ivdomain.ErrorMessage{Type: ivdomain.MsgError, Message: "context budget exceeded"})
		turn.SendQuestionOnce(s.send)
		if turn.IsTopicComplete && next == nil {
			e.SendEvaluation(ctx, s.send, s.orgID, s.interviewID)
		}
		return
	}

	if e.llm == nil {
		e.log.Error().Str("interview_id", s.interviewID.String()).Str("org_id", s.orgID).Msg("llm provider is nil")
		s.send(ivdomain.ErrorMessage{Type: ivdomain.MsgError, Message: "llm unavailable"})
		turn.SendQuestionOnce(s.send)
		if turn.IsTopicComplete && next == nil {
			e.SendEvaluation(ctx, s.send, s.orgID, s.interviewID)
		}
		return
	}

	ch, err := e.llm.ChatStream(ctx, llm.ChatRequest{OrgID: s.orgID, Messages: chatMsgs})
	if err != nil {
		e.log.Error().Err(err).Str("interview_id", s.interviewID.String()).Str("org_id", s.orgID).Msg("chat stream failed")
		s.send(ivdomain.ErrorMessage{Type: ivdomain.MsgError, Message: "llm unavailable"})
		turn.SendQuestionOnce(s.send)
		if turn.IsTopicComplete && next == nil {
			e.SendEvaluation(ctx, s.send, s.orgID, s.interviewID)
		}
		return
	}
	var final strings.Builder
	for token := range ch {
		final.WriteString(token)
		s.send(ivdomain.TokenMessage{Type: ivdomain.MsgToken, Content: token})
	}
	if ctx.Err() != nil {
		return
	}

	if !turn.IsTopicComplete {
		s.historyMu.Lock()
		s.history = append(s.history, gensvc.ContextMessage{Role: gensvc.RoleAssistant, Content: final.String()})
		s.historyMu.Unlock()
	}

	s.send(ivdomain.ResponseMessage{
		Type:            ivdomain.MsgResponse,
		Content:         final.String(),
		IsTopicComplete: turn.IsTopicComplete,
		TopicTurn:       turn.TopicTurn,
		MaxTopicTurns:   turn.MaxTopicTurns,
	})
	if turn.IsTopicComplete {
		turn.SendQuestionOnce(s.send)
		if next == nil {
			e.SendEvaluation(ctx, s.send, s.orgID, s.interviewID)
		}
	}
}

// SendEvaluation computes evaluation inline or falls back to async enqueue.
func (e *DialogueEngine) SendEvaluation(ctx context.Context, send func(any), orgID string, interviewID uuid.UUID) {
	if e.llm == nil {
		e.PendingEvaluation(send, orgID, interviewID)
		return
	}
	evalCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	pairs, err := e.svc.Transcript(evalCtx, orgID, interviewID)
	if err != nil {
		e.log.Warn().Err(err).Str("interview_id", interviewID.String()).Str("org_id", orgID).Msg("failed to load transcript for inline evaluation; deferring to worker")
		e.PendingEvaluation(send, orgID, interviewID)
		return
	}
	report, err := evalllm.NewEvaluator(e.llm).Evaluate(evalCtx, orgID, pairs)
	if err != nil {
		e.log.Warn().Err(err).Str("interview_id", interviewID.String()).Str("org_id", orgID).Msg("inline evaluation failed, deferring to worker")
		e.PendingEvaluation(send, orgID, interviewID)
		return
	}
	raw, err := json.Marshal(report)
	if err != nil {
		e.log.Error().Err(err).Str("interview_id", interviewID.String()).Str("org_id", orgID).Msg("failed to marshal evaluation report")
		e.PendingEvaluation(send, orgID, interviewID)
		return
	}
	if err := e.svc.EvaluateAndPersist(evalCtx, orgID, interviewID, raw); err != nil {
		e.log.Error().Err(err).Str("interview_id", interviewID.String()).Str("org_id", orgID).Msg("persist evaluation failed; falling back to pending")
		e.PendingEvaluation(send, orgID, interviewID)
		return
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

// PendingEvaluation enqueues retry evaluation task and notifies client of pending state.
func (e *DialogueEngine) PendingEvaluation(send func(any), orgID string, interviewID uuid.UUID) {
	if err := e.svc.EnqueueEvaluation(context.Background(), orgID, interviewID); err != nil {
		e.log.Warn().Err(err).Str("interview_id", interviewID.String()).Str("org_id", orgID).Msg("enqueue evaluation retry failed")
	}
	send(ivdomain.EvaluationMessage{
		Type:   ivdomain.MsgEvaluation,
		Scores: map[string]float64{},
		Status: ivdomain.EvalPending,
	})
}

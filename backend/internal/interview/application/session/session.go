package session

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	ivdomain "github.com/intivai/backend/internal/interview/domain"
	gensvc "github.com/intivai/backend/internal/interview/domain/service"
	sbapp "github.com/intivai/backend/internal/sandbox/application"
	sharederrors "github.com/intivai/backend/internal/shared/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func tracer() trace.Tracer {
	return otel.Tracer("github.com/intivai/backend/internal/interview/application/session")
}

const (
	MaxConcurrentCodeRuns = 4
	CodeRunTimeout        = 30 * time.Second
	QAAnswerTimeout       = 45 * time.Second
	MaxSandboxTestCases   = 20
)

// ErrorFrame maps an error to a WS domain error frame.
func ErrorFrame(err error) ivdomain.ErrorMessage {
	frame := ivdomain.ErrorMessage{Type: ivdomain.MsgError, Message: err.Error()}
	var de *sharederrors.DomainError
	if errors.As(err, &de) {
		frame.Code = de.Code
	}
	return frame
}

// QARefusal builds a polite refused qa_answer frame.
func QARefusal(question, message string) ivdomain.QAAnswerMessage {
	return ivdomain.QAAnswerMessage{Type: ivdomain.MsgQaAnswer, Question: question, Answer: message, Refused: true}
}

func isErrInterviewNotActive(err error) bool {
	var de *sharederrors.DomainError
	return errors.As(err, &de) && de.Code == "INTERVIEW_NOT_ACTIVE"
}

func safeActionAttr(action string) string {
	switch action {
	case "reply", "advance":
		return action
	default:
		return "unknown"
	}
}

// Config provides parameters to instantiate a new InterviewSession.
type Config struct {
	Ctx                  context.Context
	OrgID                string
	InterviewID          uuid.UUID
	SessionID            string
	Prompt               string
	QAContext            string
	GroundingUnavailable bool
	History              []gensvc.ContextMessage
	Send                 func(any)
	SendError            func(error)
	CodeRunner           sbapp.CodeRunner
	AcquireCodeRun       func() bool
	ReleaseCodeRun       func()
}

// InterviewSession manages connection state, turn concurrency, LLM streaming,
// debounced updates, and interview protocol handling.
type InterviewSession struct {
	ctx                  context.Context
	orgID                string
	interviewID          uuid.UUID
	sessionID            string
	prompt               string
	qaContext            string
	groundingUnavailable bool
	send                 func(any)
	sendError            func(error)
	codeRunner           sbapp.CodeRunner
	acquireCodeRun       func() bool
	releaseCodeRun       func()
	engine               *DialogueEngine

	historyMu         sync.Mutex
	history           []gensvc.ContextMessage
	lastQuestion      *ivdomain.Question
	questionInHistory bool
	archetype         string
	onQuestion        func(*ivdomain.Question)

	lastTouch    time.Time
	touchMu      sync.Mutex
	touchMuTimer *time.Timer

	signalMu     sync.Mutex
	streamCancel context.CancelFunc
	streamDone   chan struct{}

	turn       *TurnState
	turnActive bool
	turnMu     sync.Mutex

	qaMu     sync.Mutex
	qaActive bool

	codeRunMu     sync.Mutex
	codeRunActive bool
}

// NewSession initializes an InterviewSession attached to a DialogueEngine.
func (e *DialogueEngine) NewSession(cfg Config) *InterviewSession {
	s := &InterviewSession{
		ctx:                  cfg.Ctx,
		orgID:                cfg.OrgID,
		interviewID:          cfg.InterviewID,
		sessionID:            cfg.SessionID,
		prompt:               cfg.Prompt,
		qaContext:            cfg.QAContext,
		groundingUnavailable: cfg.GroundingUnavailable,
		history:              cfg.History,
		send:                 cfg.Send,
		sendError:            cfg.SendError,
		codeRunner:           cfg.CodeRunner,
		acquireCodeRun:       cfg.AcquireCodeRun,
		releaseCodeRun:       cfg.ReleaseCodeRun,
		engine:               e,
	}
	if s.sendError == nil && s.send != nil {
		s.sendError = func(err error) {
			s.send(ErrorFrame(err))
		}
	}
	s.onQuestion = func(q *ivdomain.Question) {
		s.historyMu.Lock()
		s.lastQuestion = q
		s.questionInHistory = false
		arch, _ := ivdomain.DetermineQuestionArchetype(*q)
		s.archetype = arch
		s.historyMu.Unlock()
	}
	return s
}

// SetQuestionForTest sets the current question and updates archetype for unit tests.
func (s *InterviewSession) SetQuestionForTest(q *ivdomain.Question) {
	if s.onQuestion != nil {
		s.onQuestion(q)
	}
}

// SetTurnForTest sets the internal turn state for unit testing.
func (s *InterviewSession) SetTurnForTest(t *TurnState) {
	s.turn = t
}

// SetArchetypeForTest sets the archetype for unit testing.
func (s *InterviewSession) SetArchetypeForTest(arch string) {
	s.historyMu.Lock()
	s.archetype = arch
	s.historyMu.Unlock()
}

// GetHistoryForTest returns a copy of history for unit testing.
func (s *InterviewSession) GetHistoryForTest() []gensvc.ContextMessage {
	s.historyMu.Lock()
	defer s.historyMu.Unlock()
	res := make([]gensvc.ContextMessage, len(s.history))
	copy(res, s.history)
	return res
}

// SendStartAndQuestion delivers the interview start frame and current question.
func (s *InterviewSession) SendStartAndQuestion() {
	next, total, status, err := s.engine.svc.CurrentState(s.ctx, s.orgID, s.interviewID)
	if err != nil {
		s.engine.log.Error().Err(err).Str("interview_id", s.interviewID.String()).Str("org_id", s.orgID).Msg("failed to load current state for start frame")
		s.sendError(err)
		return
	}
	remSec := s.engine.svc.SessionRemaining(s.ctx, s.orgID, s.interviewID)
	s.send(ivdomain.InterviewStartMessage{
		Type:             ivdomain.MsgStart,
		SessionID:        s.sessionID,
		TotalQuestions:   total,
		SessionBudgetSec: int(ivdomain.MaxInterviewDuration.Seconds()),
	})
	if status == ivdomain.StatusInProgress && next != nil {
		arch, limit := ivdomain.DetermineQuestionArchetype(*next)
		s.send(ivdomain.QuestionMessage{
			Type:                ivdomain.MsgQuestion,
			Content:             next.Content,
			Idx:                 next.Idx,
			TotalQuestions:      total,
			IsProbe:             next.IsProbe,
			Archetype:           arch,
			TimeLimitSec:        limit,
			SessionRemainingSec: remSec,
			TopicTurn:           1,
			MaxTopicTurns:       ivdomain.MaxTurnsPerTopic,
		})
		if s.onQuestion != nil {
			s.onQuestion(next)
		}
	}
}

// HandleResume handles client reconnection/resume.
func (s *InterviewSession) HandleResume() {
	s.signalMu.Lock()
	if s.streamCancel != nil {
		s.streamCancel()
	}
	if s.streamDone != nil {
		<-s.streamDone
		s.streamDone = nil
	}
	s.streamCancel = nil
	s.turn = nil
	s.signalMu.Unlock()
	s.SendStartAndQuestion()
}

// HandleInterrupt stops any active stream and forces next question dispatch.
func (s *InterviewSession) HandleInterrupt() {
	s.signalMu.Lock()
	if s.streamCancel != nil {
		s.streamCancel()
	}
	if s.streamDone != nil {
		<-s.streamDone
		s.streamDone = nil
	}
	s.streamCancel = nil
	s.signalMu.Unlock()

	s.send(ivdomain.ResponseMessage{Type: ivdomain.MsgResponse, Content: "Interrupted."})
	if s.turn == nil {
		return
	}
	if s.turn.WasQuestionSent() {
		s.turn = nil
		return
	}
	if !s.turn.IsTopicComplete {
		res, err := s.engine.svc.ProcessTopicDialogue(s.ctx, s.orgID, s.interviewID, "The candidate interrupted the AI response.", "advance", nil)
		if err != nil {
			s.sendError(err)
			s.turn = nil
			return
		}
		if res.TransitionErr != nil {
			s.sendError(res.TransitionErr)
		}
		s.turn = &TurnState{
			Next:            res.NextQuestion,
			Total:           res.TotalQuestions,
			RemainingSec:    s.engine.svc.SessionRemaining(s.ctx, s.orgID, s.interviewID),
			IsTopicComplete: res.IsTopicComplete,
			TopicTurn:       res.CurrentTurn,
			MaxTopicTurns:   res.MaxTurns,
			OnSent:          s.onQuestion,
		}
	}
	if s.turn.Next != nil {
		s.turn.SendQuestionOnce(s.send)
	} else if s.turn.IsTopicComplete {
		s.engine.SendEvaluation(s.ctx, s.send, s.orgID, s.interviewID)
	}
	s.turn = nil
}

// DebouncedTouch records activity with a 500ms debounce.
func (s *InterviewSession) DebouncedTouch() {
	s.touchMu.Lock()
	now := time.Now()
	if now.Sub(s.lastTouch) > 500*time.Millisecond && s.touchMuTimer == nil {
		s.lastTouch = now
		s.touchMu.Unlock()
		if err := s.engine.svc.TouchInterview(s.ctx, s.orgID, s.interviewID); err != nil {
			s.engine.log.Warn().Err(err).Str("interview_id", s.interviewID.String()).Str("org_id", s.orgID).Msg("interview touch failed")
		}
		return
	}
	if s.touchMuTimer != nil {
		s.touchMuTimer.Stop()
	}
	s.touchMuTimer = time.AfterFunc(500*time.Millisecond, func() {
		s.touchMu.Lock()
		s.touchMuTimer = nil
		s.lastTouch = time.Now()
		s.touchMu.Unlock()
		if err := s.engine.svc.TouchInterview(s.ctx, s.orgID, s.interviewID); err != nil {
			s.engine.log.Warn().Err(err).Str("interview_id", s.interviewID.String()).Str("org_id", s.orgID).Msg("debounced interview touch failed")
		}
	})
	s.touchMu.Unlock()
}

// HandleTelemetry records candidate telemetry proctoring events.
func (s *InterviewSession) HandleTelemetry(m ivdomain.TelemetryMessage) {
	var evTime time.Time
	if m.Timestamp != "" {
		var err error
		evTime, err = time.Parse(time.RFC3339, m.Timestamp)
		if err != nil {
			s.engine.log.Warn().Err(err).Str("interview_id", s.interviewID.String()).Str("org_id", s.orgID).Msg("invalid telemetry timestamp; using server time")
		}
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
	if err := s.engine.svc.RecordTelemetry(s.ctx, s.orgID, s.interviewID, event); err != nil {
		s.engine.log.Error().Err(err).Str("interview_id", s.interviewID.String()).Str("org_id", s.orgID).Msg("record telemetry failed")
	}
}

// HandleAnswer processes candidate answers and initiates streaming response.
func (s *InterviewSession) HandleAnswer(m ivdomain.AnswerMessage) bool {
	s.turnMu.Lock()
	if s.turnActive {
		s.turnMu.Unlock()
		s.sendError(sharederrors.NewDomainError(ivdomain.ErrCodeTurnInProgress, "previous answer is still being processed; wait for the response or interrupt"))
		return true
	}
	s.turnActive = true
	s.turnMu.Unlock()

	answerCtx, answerSpan := tracer().Start(s.ctx, "interview.answer.process",
		trace.WithAttributes(
			attribute.String("org.id", s.orgID),
			attribute.String("interview.id", s.interviewID.String()),
			attribute.String("interview.action", safeActionAttr(m.Action)),
		))
	res, err := s.engine.svc.ProcessTopicDialogue(answerCtx, s.orgID, s.interviewID, m.Content, m.Action, m.PacingTelemetry)
	if err != nil {
		answerSpan.RecordError(err)
		answerSpan.SetStatus(codes.Error, err.Error())
		answerSpan.End()
		s.turnMu.Lock()
		s.turnActive = false
		s.turnMu.Unlock()
		s.sendError(err)
		return false
	}
	answerSpan.End()

	if res.TransitionErr != nil {
		s.sendError(res.TransitionErr)
	}

	s.historyMu.Lock()
	if s.lastQuestion != nil && !s.questionInHistory {
		s.history = append(s.history,
			gensvc.ContextMessage{Role: gensvc.RoleAssistant, Content: s.lastQuestion.Content},
			gensvc.ContextMessage{Role: gensvc.RoleUser, Content: m.Content},
		)
		s.questionInHistory = true
	} else {
		s.history = append(s.history, gensvc.ContextMessage{Role: gensvc.RoleUser, Content: m.Content})
	}
	s.historyMu.Unlock()

	remSec := s.engine.svc.SessionRemaining(s.ctx, s.orgID, s.interviewID)
	s.turn = &TurnState{
		Next:            res.NextQuestion,
		Total:           res.TotalQuestions,
		RemainingSec:    remSec,
		IsTopicComplete: res.IsTopicComplete,
		TopicTurn:       res.CurrentTurn,
		MaxTopicTurns:   res.MaxTurns,
		OnSent:          s.onQuestion,
	}

	streamCtx, cancelStream := context.WithCancel(answerCtx)
	streamDone := make(chan struct{})
	s.signalMu.Lock()
	s.streamCancel = cancelStream
	s.streamDone = streamDone
	s.signalMu.Unlock()

	go func() {
		defer close(streamDone)
		defer func() {
			s.turnMu.Lock()
			s.turnActive = false
			s.turnMu.Unlock()
		}()
		s.engine.StreamAndRespond(streamCtx, s, m.Content, res.NextQuestion, s.turn)
	}()
	return true
}

// ReserveCodeRun claims single in-flight per-session code execution slot.
func (s *InterviewSession) ReserveCodeRun() bool {
	s.codeRunMu.Lock()
	defer s.codeRunMu.Unlock()
	if s.codeRunActive {
		return false
	}
	s.codeRunActive = true
	return true
}

// ReleaseCodeRun frees per-session code execution slot.
func (s *InterviewSession) ReleaseCodeRun() {
	s.codeRunMu.Lock()
	s.codeRunActive = false
	s.codeRunMu.Unlock()
}

// Close releases resources, cancels pending touch timer, and stops active stream.
func (s *InterviewSession) Close() {
	s.touchMu.Lock()
	if s.touchMuTimer != nil {
		s.touchMuTimer.Stop()
		s.touchMuTimer = nil
	}
	s.touchMu.Unlock()

	s.signalMu.Lock()
	if s.streamCancel != nil {
		s.streamCancel()
		s.streamCancel = nil
	}
	s.signalMu.Unlock()
}

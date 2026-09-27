package session_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	ivapp "github.com/intivai/backend/internal/interview/application"
	"github.com/intivai/backend/internal/interview/application/session"
	ivdomain "github.com/intivai/backend/internal/interview/domain"
	"github.com/intivai/backend/internal/llm"
	sbdomain "github.com/intivai/backend/internal/sandbox/domain"
	"github.com/rs/zerolog"
)

type blockingLLM struct {
	release chan struct{}
}

func (b *blockingLLM) Chat(_ context.Context, _ llm.ChatRequest) (*llm.ChatResponse, error) {
	return &llm.ChatResponse{Content: "ok"}, nil
}

func (b *blockingLLM) ChatStream(_ context.Context, _ llm.ChatRequest) (<-chan string, error) {
	ch := make(chan string)
	go func() {
		<-b.release
		close(ch)
	}()
	return ch, nil
}

func (b *blockingLLM) StructuredOutput(_ context.Context, _ llm.StructuredRequest) (any, error) {
	return nil, nil
}

func (b *blockingLLM) Embed(_ context.Context, _ string) ([]float32, error) {
	return nil, nil
}

func (b *blockingLLM) CountTokens(text string) int {
	return len(text) / 4
}

type staticLLM struct {
	content string
	tokens  []string
	err     error
}

func (s *staticLLM) Chat(_ context.Context, _ llm.ChatRequest) (*llm.ChatResponse, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &llm.ChatResponse{Content: s.content}, nil
}

func (s *staticLLM) ChatStream(_ context.Context, _ llm.ChatRequest) (<-chan string, error) {
	if s.err != nil {
		return nil, s.err
	}
	ch := make(chan string, len(s.tokens))
	for _, tok := range s.tokens {
		ch <- tok
	}
	close(ch)
	return ch, nil
}

func (s *staticLLM) StructuredOutput(_ context.Context, _ llm.StructuredRequest) (any, error) {
	return nil, nil
}

func (s *staticLLM) Embed(_ context.Context, _ string) ([]float32, error) {
	return nil, nil
}

func (s *staticLLM) CountTokens(text string) int {
	return len(text) / 4
}

type mockService struct {
	mu               sync.Mutex
	sessionRemaining int
	touchCount       int
	recordedEvents   []ivdomain.ProctoringEvent
	recordedQAs      [][2]string
	qaRemaining      int
	qaRemainingErr   error
	recordQAErr      error
	question         *ivdomain.Question
	total            int
	status           ivdomain.Status
	currentStateErr  error
	transcriptPairs  []ivdomain.TranscriptPair
	transcriptErr    error
	evalPersisted    []byte
	persistErr       error
	enqueuedEvals    int
	codingSession    *ivdomain.CodingSession
}

func (m *mockService) GetCodingSession() *ivdomain.CodingSession {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.codingSession
}

func (m *mockService) GetTouchCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.touchCount
}

func (m *mockService) GetRecordedEvents() []ivdomain.ProctoringEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([]ivdomain.ProctoringEvent, len(m.recordedEvents))
	copy(cp, m.recordedEvents)
	return cp
}

func (m *mockService) GetRecordedQAs() [][2]string {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([][2]string, len(m.recordedQAs))
	copy(cp, m.recordedQAs)
	return cp
}

func (m *mockService) GetEnqueuedEvals() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.enqueuedEvals
}

func (m *mockService) ProcessTopicDialogue(_ context.Context, _ string, _ uuid.UUID, _ string, _ string, _ *ivdomain.PacingMetrics) (*ivapp.TopicDialogueResult, error) {
	time.Sleep(50 * time.Millisecond)
	return &ivapp.TopicDialogueResult{
		CurrentTurn:     1,
		MaxTurns:        3,
		IsTopicComplete: false,
	}, nil
}

func (m *mockService) CandidateQARemaining(_ context.Context, _ string, _ uuid.UUID) (int, error) {
	if m.qaRemainingErr != nil {
		return 0, m.qaRemainingErr
	}
	return m.qaRemaining, nil
}

func (m *mockService) SessionRemaining(_ context.Context, _ string, _ uuid.UUID) int {
	return m.sessionRemaining
}

func (m *mockService) TouchInterview(_ context.Context, _ string, _ uuid.UUID) error {
	m.mu.Lock()
	m.touchCount++
	m.mu.Unlock()
	return nil
}

func (m *mockService) RecordTelemetry(_ context.Context, _ string, _ uuid.UUID, event ivdomain.ProctoringEvent) error {
	m.mu.Lock()
	m.recordedEvents = append(m.recordedEvents, event)
	m.mu.Unlock()
	return nil
}

func (m *mockService) RecordCandidateQA(_ context.Context, _ string, _ uuid.UUID, question, answer string) error {
	if m.recordQAErr != nil {
		return m.recordQAErr
	}
	m.mu.Lock()
	m.recordedQAs = append(m.recordedQAs, [2]string{question, answer})
	m.mu.Unlock()
	return nil
}

func (m *mockService) CurrentState(_ context.Context, _ string, _ uuid.UUID) (*ivdomain.Question, int, ivdomain.Status, error) {
	if m.currentStateErr != nil {
		return nil, 0, "", m.currentStateErr
	}
	return m.question, m.total, m.status, nil
}

func (m *mockService) Transcript(_ context.Context, _ string, _ uuid.UUID) ([]ivdomain.TranscriptPair, error) {
	if m.transcriptErr != nil {
		return nil, m.transcriptErr
	}
	return m.transcriptPairs, nil
}

func (m *mockService) EvaluateAndPersist(_ context.Context, _ string, _ uuid.UUID, reportJSON []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.persistErr != nil {
		return m.persistErr
	}
	m.evalPersisted = reportJSON
	return nil
}

func (m *mockService) EnqueueEvaluation(_ context.Context, _ string, _ uuid.UUID) error {
	m.mu.Lock()
	m.enqueuedEvals++
	m.mu.Unlock()
	return nil
}

func (m *mockService) RecordCodingSession(_ context.Context, _ string, _ uuid.UUID, sess ivdomain.CodingSession) error {
	m.mu.Lock()
	m.codingSession = &sess
	m.mu.Unlock()
	return nil
}

type mockCodeRunner struct {
	result *sbdomain.ExecutionResult
	err    error
}

func (m *mockCodeRunner) Execute(_ context.Context, _ sbdomain.ExecutionRequest) (*sbdomain.ExecutionResult, error) {
	return m.result, m.err
}

func TestInterviewSession_ReserveAndReleaseQA(t *testing.T) {
	engine := session.NewDialogueEngine(&mockService{}, nil, zerolog.Nop())
	sess := engine.NewSession(session.Config{
		Ctx: context.Background(),
	})

	if !sess.ReserveQA() {
		t.Fatal("expected first ReserveQA to succeed")
	}

	if sess.ReserveQA() {
		t.Fatal("expected concurrent ReserveQA to fail")
	}

	sess.ReleaseQA()

	if !sess.ReserveQA() {
		t.Fatal("expected ReserveQA after release to succeed")
	}
}

func TestInterviewSession_HandleCodeRun_RejectsNonCodingArchetype(t *testing.T) {
	frames := []any{}
	send := func(f any) {
		frames = append(frames, f)
	}

	engine := session.NewDialogueEngine(&mockService{}, nil, zerolog.Nop())
	sess := engine.NewSession(session.Config{
		Ctx:  context.Background(),
		Send: send,
	})

	sess.HandleCodeRun(ivdomain.CodeRunMessage{
		Type:        ivdomain.MsgCodeRun,
		QuestionIdx: 1,
		Language:    "go",
		Code:        "package main",
	})

	if len(frames) != 1 {
		t.Fatalf("expected 1 frame sent, got %d", len(frames))
	}
	res, ok := frames[0].(ivdomain.CodeResultMessage)
	if !ok {
		t.Fatalf("expected CodeResultMessage, got %T", frames[0])
	}
	if res.Error != "code execution is only allowed on coding challenges" {
		t.Errorf("unexpected error message: %s", res.Error)
	}
}

func TestInterviewSession_HandleCodeRun_ExecutesCodingChallenge(t *testing.T) {
	var mu sync.Mutex
	frames := []any{}
	send := func(f any) {
		mu.Lock()
		frames = append(frames, f)
		mu.Unlock()
	}

	svc := &mockService{}
	runner := &mockCodeRunner{
		result: &sbdomain.ExecutionResult{
			Stdout:    "Hello World\n",
			ExitCode:  0,
			AllPassed: true,
		},
	}
	engine := session.NewDialogueEngine(svc, nil, zerolog.Nop())
	sess := engine.NewSession(session.Config{
		Ctx:         context.Background(),
		OrgID:       uuid.NewString(),
		InterviewID: uuid.New(),
		Send:        send,
		CodeRunner:  runner,
	})

	// Set archetype to coding
	sess.SetQuestionForTest(&ivdomain.Question{
		Idx:     1,
		Content: "Write a function in Go to reverse a string",
	})

	sess.HandleCodeRun(ivdomain.CodeRunMessage{
		Type:        ivdomain.MsgCodeRun,
		QuestionIdx: 1,
		Language:    "go",
		Code:        "package main\nfunc main() {}",
	})

	time.Sleep(20 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if len(frames) != 1 {
		t.Fatalf("expected 1 frame sent, got %d: %+v", len(frames), frames)
	}
	res, ok := frames[0].(ivdomain.CodeResultMessage)
	if !ok {
		t.Fatalf("expected CodeResultMessage, got %T", frames[0])
	}
	if res.Stdout != "Hello World\n" {
		t.Errorf("expected stdout 'Hello World\\n', got %q", res.Stdout)
	}
	if svc.GetCodingSession() == nil {
		t.Error("expected RecordCodingSession to be called")
	}
}

func TestInterviewSession_HandleInterrupt_SendsInterruptedResponse(t *testing.T) {
	frames := []any{}
	send := func(f any) {
		frames = append(frames, f)
	}

	engine := session.NewDialogueEngine(&mockService{}, nil, zerolog.Nop())
	sess := engine.NewSession(session.Config{
		Ctx:  context.Background(),
		Send: send,
	})

	sess.HandleInterrupt()

	if len(frames) != 1 {
		t.Fatalf("expected 1 frame sent, got %d", len(frames))
	}
	resp, ok := frames[0].(ivdomain.ResponseMessage)
	if !ok {
		t.Fatalf("expected ResponseMessage, got %T", frames[0])
	}
	if resp.Content != "Interrupted." {
		t.Errorf("expected 'Interrupted.', got: %s", resp.Content)
	}
}

func TestInterviewSession_HandleAnswer_OverlappingTurnRejected(t *testing.T) {
	var mu sync.Mutex
	frames := []any{}
	send := func(f any) {
		mu.Lock()
		frames = append(frames, f)
		mu.Unlock()
	}

	release := make(chan struct{})
	defer close(release)
	mockLLM := &blockingLLM{release: release}

	engine := session.NewDialogueEngine(&mockService{}, mockLLM, zerolog.Nop())
	sess := engine.NewSession(session.Config{
		Ctx:         context.Background(),
		OrgID:       uuid.NewString(),
		InterviewID: uuid.New(),
		Prompt:      "system prompt",
		Send:        send,
	})

	// Start first turn
	_ = sess.HandleAnswer(ivdomain.AnswerMessage{
		Idx:     1,
		Content: "My answer",
	})

	// Immediate second answer while stream is in flight must be rejected with TURN_IN_PROGRESS
	sess.HandleAnswer(ivdomain.AnswerMessage{
		Idx:     1,
		Content: "Another answer while first is running",
	})

	time.Sleep(10 * time.Millisecond)

	mu.Lock()
	var foundTurnInProgress bool
	for _, f := range frames {
		if em, ok := f.(ivdomain.ErrorMessage); ok && em.Code == ivdomain.ErrCodeTurnInProgress {
			foundTurnInProgress = true
			break
		}
	}
	mu.Unlock()

	if !foundTurnInProgress {
		mu.Lock()
		t.Fatalf("expected TURN_IN_PROGRESS error frame, got: %+v", frames)
		mu.Unlock()
	}
}

func TestInterviewSession_SendStartAndQuestion(t *testing.T) {
	frames := []any{}
	send := func(f any) {
		frames = append(frames, f)
	}

	svc := &mockService{
		status:           ivdomain.StatusInProgress,
		total:            3,
		sessionRemaining: 1800,
		question: &ivdomain.Question{
			Idx:      1,
			Content:  "Explain database normalization",
			Category: "database",
		},
	}
	engine := session.NewDialogueEngine(svc, nil, zerolog.Nop())
	sess := engine.NewSession(session.Config{
		Ctx:         context.Background(),
		OrgID:       uuid.NewString(),
		InterviewID: uuid.New(),
		SessionID:   "sess-123",
		Send:        send,
	})

	sess.SendStartAndQuestion()

	if len(frames) != 2 {
		t.Fatalf("expected 2 frames (start + question), got %d: %+v", len(frames), frames)
	}
	startMsg, ok := frames[0].(ivdomain.InterviewStartMessage)
	if !ok || startMsg.Type != ivdomain.MsgStart || startMsg.TotalQuestions != 3 {
		t.Errorf("unexpected start frame: %+v", frames[0])
	}
	qMsg, ok := frames[1].(ivdomain.QuestionMessage)
	if !ok || qMsg.Type != ivdomain.MsgQuestion || qMsg.Content != "Explain database normalization" {
		t.Errorf("unexpected question frame: %+v", frames[1])
	}
}

func TestInterviewSession_HandleResume(t *testing.T) {
	frames := []any{}
	send := func(f any) {
		frames = append(frames, f)
	}

	svc := &mockService{
		status:           ivdomain.StatusInProgress,
		total:            3,
		sessionRemaining: 1500,
		question: &ivdomain.Question{
			Idx:      2,
			Content:  "What is Raft consensus?",
			Category: "distributed_systems",
		},
	}
	engine := session.NewDialogueEngine(svc, nil, zerolog.Nop())
	sess := engine.NewSession(session.Config{
		Ctx:         context.Background(),
		OrgID:       uuid.NewString(),
		InterviewID: uuid.New(),
		SessionID:   "sess-456",
		Send:        send,
	})

	sess.HandleResume()

	if len(frames) != 2 {
		t.Fatalf("expected 2 frames on resume, got %d", len(frames))
	}
}

func TestInterviewSession_DebouncedTouch(t *testing.T) {
	svc := &mockService{}
	engine := session.NewDialogueEngine(svc, nil, zerolog.Nop())
	sess := engine.NewSession(session.Config{
		Ctx:         context.Background(),
		OrgID:       uuid.NewString(),
		InterviewID: uuid.New(),
	})

	sess.DebouncedTouch()
	sess.DebouncedTouch()
	sess.DebouncedTouch()

	if svc.GetTouchCount() != 1 {
		t.Errorf("expected debounced touch count 1, got %d", svc.GetTouchCount())
	}
}

func TestInterviewSession_HandleTelemetry(t *testing.T) {
	svc := &mockService{}
	engine := session.NewDialogueEngine(svc, nil, zerolog.Nop())
	sess := engine.NewSession(session.Config{
		Ctx:         context.Background(),
		OrgID:       uuid.NewString(),
		InterviewID: uuid.New(),
	})

	sess.HandleTelemetry(ivdomain.TelemetryMessage{
		Type:      ivdomain.MsgTelemetry,
		EventType: "tab_switch",
	})

	events := svc.GetRecordedEvents()
	if len(events) != 1 || string(events[0].Type) != "tab_switch" {
		t.Errorf("unexpected recorded telemetry: %+v", events)
	}
}

func TestInterviewSession_HandleCandidateQuestion(t *testing.T) {
	t.Run("QuotaExceeded", func(t *testing.T) {
		var mu sync.Mutex
		frames := []any{}
		send := func(f any) {
			mu.Lock()
			frames = append(frames, f)
			mu.Unlock()
		}

		svc := &mockService{qaRemaining: 0}
		engine := session.NewDialogueEngine(svc, nil, zerolog.Nop())
		sess := engine.NewSession(session.Config{
			Ctx:         context.Background(),
			OrgID:       uuid.NewString(),
			InterviewID: uuid.New(),
			Send:        send,
		})

		sess.HandleCandidateQuestion(ivdomain.CandidateQuestionMessage{
			Type:    ivdomain.MsgCandidateQuestion,
			Content: "What is the team size?",
		})

		time.Sleep(20 * time.Millisecond)

		mu.Lock()
		defer mu.Unlock()
		if len(frames) != 1 {
			t.Fatalf("expected 1 frame, got %d", len(frames))
		}
		refusal, ok := frames[0].(ivdomain.QAAnswerMessage)
		if !ok || !refusal.Refused {
			t.Fatalf("expected refused QAAnswerMessage, got %+v", frames[0])
		}
	})

	t.Run("GroundingUnavailable", func(t *testing.T) {
		var mu sync.Mutex
		frames := []any{}
		send := func(f any) {
			mu.Lock()
			frames = append(frames, f)
			mu.Unlock()
		}

		svc := &mockService{qaRemaining: 3}
		engine := session.NewDialogueEngine(svc, nil, zerolog.Nop())
		sess := engine.NewSession(session.Config{
			Ctx:                  context.Background(),
			OrgID:                uuid.NewString(),
			InterviewID:          uuid.New(),
			GroundingUnavailable: true,
			Send:                 send,
		})

		sess.HandleCandidateQuestion(ivdomain.CandidateQuestionMessage{
			Type:    ivdomain.MsgCandidateQuestion,
			Content: "What are your benefits?",
		})

		time.Sleep(20 * time.Millisecond)

		mu.Lock()
		defer mu.Unlock()
		if len(frames) != 1 {
			t.Fatalf("expected 1 frame, got %d", len(frames))
		}
		refusal, ok := frames[0].(ivdomain.QAAnswerMessage)
		if !ok || !refusal.Refused {
			t.Fatalf("expected refused QAAnswerMessage, got %+v", frames[0])
		}
	})

	t.Run("SuccessfulAnswer", func(t *testing.T) {
		var mu sync.Mutex
		frames := []any{}
		send := func(f any) {
			mu.Lock()
			frames = append(frames, f)
			mu.Unlock()
		}

		svc := &mockService{qaRemaining: 3}
		mockL := &staticLLM{content: "We have flexible working hours."}
		engine := session.NewDialogueEngine(svc, mockL, zerolog.Nop())
		sess := engine.NewSession(session.Config{
			Ctx:         context.Background(),
			OrgID:       uuid.NewString(),
			InterviewID: uuid.New(),
			QAContext:   "Work culture details",
			Send:        send,
		})

		sess.HandleCandidateQuestion(ivdomain.CandidateQuestionMessage{
			Type:    ivdomain.MsgCandidateQuestion,
			Content: "What are the hours?",
		})

		time.Sleep(20 * time.Millisecond)

		mu.Lock()
		defer mu.Unlock()
		if len(frames) != 1 {
			t.Fatalf("expected 1 frame, got %d: %+v", len(frames), frames)
		}
		ans, ok := frames[0].(ivdomain.QAAnswerMessage)
		if !ok || ans.Answer != "We have flexible working hours." {
			t.Fatalf("unexpected answer frame: %+v", frames[0])
		}
		recorded := svc.GetRecordedQAs()
		if len(recorded) != 1 {
			t.Errorf("expected 1 recorded QA, got %d", len(recorded))
		}
	})
	t.Run("NilLLMProvider", func(t *testing.T) {
		var mu sync.Mutex
		frames := []any{}
		send := func(f any) {
			mu.Lock()
			frames = append(frames, f)
			mu.Unlock()
		}

		svc := &mockService{qaRemaining: 3}
		engine := session.NewDialogueEngine(svc, nil, zerolog.Nop())
		sess := engine.NewSession(session.Config{
			Ctx:         context.Background(),
			OrgID:       uuid.NewString(),
			InterviewID: uuid.New(),
			QAContext:   "Work culture details",
			Send:        send,
		})

		sess.HandleCandidateQuestion(ivdomain.CandidateQuestionMessage{
			Type:    ivdomain.MsgCandidateQuestion,
			Content: "What are the hours?",
		})

		time.Sleep(20 * time.Millisecond)

		mu.Lock()
		defer mu.Unlock()
		if len(frames) != 1 {
			t.Fatalf("expected 1 frame, got %d", len(frames))
		}
		refusal, ok := frames[0].(ivdomain.QAAnswerMessage)
		if !ok || !refusal.Refused {
			t.Fatalf("expected refused QAAnswerMessage when LLM is nil, got %+v", frames[0])
		}
	})
}

func TestDialogueEngine_PendingEvaluation_And_TranscriptError(t *testing.T) {
	frames := []any{}
	send := func(f any) {
		frames = append(frames, f)
	}

	svc := &mockService{
		transcriptErr: errors.New("db error"),
	}
	engine := session.NewDialogueEngine(svc, nil, zerolog.Nop())

	engine.SendEvaluation(context.Background(), send, uuid.NewString(), uuid.New())

	if len(frames) != 1 {
		t.Fatalf("expected 1 frame, got %d", len(frames))
	}
	evalMsg, ok := frames[0].(ivdomain.EvaluationMessage)
	if !ok || evalMsg.Status != ivdomain.EvalPending {
		t.Errorf("expected EvalPending status, got: %+v", frames[0])
	}
	if svc.GetEnqueuedEvals() != 1 {
		t.Errorf("expected 1 enqueued eval, got %d", svc.GetEnqueuedEvals())
	}
}

func TestDialogueEngine_SendEvaluation_NilLLM_FallsBackToPending(t *testing.T) {
	frames := []any{}
	send := func(f any) {
		frames = append(frames, f)
	}

	svc := &mockService{}
	engine := session.NewDialogueEngine(svc, nil, zerolog.Nop())

	engine.SendEvaluation(context.Background(), send, uuid.NewString(), uuid.New())

	if len(frames) != 1 {
		t.Fatalf("expected 1 frame, got %d", len(frames))
	}
	evalMsg, ok := frames[0].(ivdomain.EvaluationMessage)
	if !ok || evalMsg.Status != ivdomain.EvalPending {
		t.Errorf("expected EvalPending status on nil LLM, got: %+v", frames[0])
	}
	if svc.GetEnqueuedEvals() != 1 {
		t.Errorf("expected 1 enqueued eval, got %d", svc.GetEnqueuedEvals())
	}
}

func TestSession_HandleInterrupt_FinalQuestionTriggersEvaluation(t *testing.T) {
	frames := []any{}
	send := func(f any) {
		frames = append(frames, f)
	}

	svc := &mockService{}
	engine := session.NewDialogueEngine(svc, nil, zerolog.Nop())
	sess := engine.NewSession(session.Config{
		Ctx:         context.Background(),
		OrgID:       uuid.NewString(),
		InterviewID: uuid.New(),
		Send:        send,
	})

	sess.SetTurnForTest(&session.TurnState{
		Next:            nil,
		Total:           5,
		IsTopicComplete: true,
	})

	sess.HandleInterrupt()

	if len(frames) < 2 {
		t.Fatalf("expected at least 2 frames (interrupted + eval), got %d: %+v", len(frames), frames)
	}
	evalMsg, ok := frames[1].(ivdomain.EvaluationMessage)
	if !ok {
		t.Fatalf("expected evaluation frame as second message, got: %+v", frames[1])
	}
	if evalMsg.Status != ivdomain.EvalPending {
		t.Errorf("expected EvalPending status, got %s", evalMsg.Status)
	}
}

func TestSession_HandleCodeRun_SingleFlightPerSession(t *testing.T) {
	frames := []any{}
	var mu sync.Mutex
	send := func(f any) {
		mu.Lock()
		frames = append(frames, f)
		mu.Unlock()
	}

	svc := &mockService{}
	engine := session.NewDialogueEngine(svc, nil, zerolog.Nop())
	sess := engine.NewSession(session.Config{
		Ctx:         context.Background(),
		OrgID:       uuid.NewString(),
		InterviewID: uuid.New(),
		Send:        send,
	})
	sess.SetArchetypeForTest(ivdomain.ArchetypeCoding)

	if !sess.ReserveCodeRun() {
		t.Fatal("expected first code run reservation to succeed")
	}
	if sess.ReserveCodeRun() {
		t.Fatal("expected second concurrent code run reservation to fail")
	}

	sess.HandleCodeRun(ivdomain.CodeRunMessage{
		Type:     ivdomain.MsgCodeRun,
		Language: "python",
		Code:     "print(1)",
	})

	mu.Lock()
	defer mu.Unlock()
	if len(frames) != 1 {
		t.Fatalf("expected 1 frame, got %d", len(frames))
	}
	res, ok := frames[0].(ivdomain.CodeResultMessage)
	if !ok || res.Error == "" {
		t.Fatalf("expected error code result frame, got %+v", frames[0])
	}
	sess.ReleaseCodeRun()
}

func TestSession_Close_StopsTimerAndStream(t *testing.T) {
	svc := &mockService{}
	engine := session.NewDialogueEngine(svc, nil, zerolog.Nop())
	sess := engine.NewSession(session.Config{
		Ctx:         context.Background(),
		OrgID:       uuid.NewString(),
		InterviewID: uuid.New(),
	})

	sess.DebouncedTouch()
	sess.Close()
	sess.Close()
}

func TestDialogueEngine_SendEvaluation_EvaluateAndPersistError_FallsBackToPending(t *testing.T) {
	frames := []any{}
	send := func(f any) {
		frames = append(frames, f)
	}

	evaluatorJSON := `{"dimensions":{"Technical":{"score":85.0}},"overall_score":85.0,"recommendation":"Strong"}`
	fakeLLM := &staticLLM{content: evaluatorJSON}

	svc := &mockService{
		persistErr: errors.New("db connection failure"),
	}
	engine := session.NewDialogueEngine(svc, fakeLLM, zerolog.Nop())

	engine.SendEvaluation(context.Background(), send, uuid.NewString(), uuid.New())

	if len(frames) != 1 {
		t.Fatalf("expected 1 frame, got %d", len(frames))
	}
	evalMsg, ok := frames[0].(ivdomain.EvaluationMessage)
	if !ok || evalMsg.Status != ivdomain.EvalPending {
		t.Errorf("expected EvalPending status when persist fails, got: %+v", frames[0])
	}
	if svc.GetEnqueuedEvals() != 1 {
		t.Errorf("expected 1 enqueued retry eval, got %d", svc.GetEnqueuedEvals())
	}
}

func TestDialogueEngine_StreamAndRespond_LLMErrorOnFinalQuestion_TriggersEvaluation(t *testing.T) {
	frames := []any{}
	var mu sync.Mutex
	send := func(f any) {
		mu.Lock()
		frames = append(frames, f)
		mu.Unlock()
	}

	svc := &mockService{}
	engine := session.NewDialogueEngine(svc, nil, zerolog.Nop())
	sess := engine.NewSession(session.Config{
		Ctx:         context.Background(),
		OrgID:       uuid.NewString(),
		InterviewID: uuid.New(),
		Send:        send,
	})

	turn := &session.TurnState{
		Next:            nil,
		Total:           3,
		IsTopicComplete: true,
	}

	engine.StreamAndRespond(context.Background(), sess, "Final answer", nil, turn)

	mu.Lock()
	defer mu.Unlock()
	if len(frames) < 2 {
		t.Fatalf("expected at least 2 frames (error + eval), got %d: %+v", len(frames), frames)
	}
	lastFrame := frames[len(frames)-1]
	evalMsg, ok := lastFrame.(ivdomain.EvaluationMessage)
	if !ok {
		t.Fatalf("expected final frame to be EvaluationMessage, got: %+v", lastFrame)
	}
	if evalMsg.Status != ivdomain.EvalPending {
		t.Errorf("expected EvalPending status on LLM error on final question, got %s", evalMsg.Status)
	}
}

func TestSession_MultiTurnDialogue_DoesNotDuplicateQuestionPromptInHistory(t *testing.T) {
	svc := &mockService{
		question: &ivdomain.Question{Idx: 1, Content: "What is a goroutine?"},
		total:    3,
	}
	fakeLLM := &staticLLM{tokens: []string{"Can you elaborate?"}}
	engine := session.NewDialogueEngine(svc, fakeLLM, zerolog.Nop())

	sess := engine.NewSession(session.Config{
		Ctx:         context.Background(),
		OrgID:       uuid.NewString(),
		InterviewID: uuid.New(),
		Send:        func(any) {},
	})
	sess.SetQuestionForTest(&ivdomain.Question{Idx: 1, Content: "What is a goroutine?"})

	sess.HandleAnswer(ivdomain.AnswerMessage{Type: ivdomain.MsgAnswer, Content: "It's a lightweight thread"})
	time.Sleep(50 * time.Millisecond)

	sess.HandleAnswer(ivdomain.AnswerMessage{Type: ivdomain.MsgAnswer, Content: "Managed by Go runtime"})
	time.Sleep(50 * time.Millisecond)

	history := sess.GetHistoryForTest()
	qCount := 0
	for _, m := range history {
		if m.Content == "What is a goroutine?" {
			qCount++
		}
	}
	if qCount != 1 {
		t.Fatalf("expected question to appear exactly once in history, got %d times in history: %+v", qCount, history)
	}
}

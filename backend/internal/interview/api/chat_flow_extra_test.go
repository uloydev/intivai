package api

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	ctxapp "github.com/intivai/backend/internal/context/application"
	ctxrepo "github.com/intivai/backend/internal/context/infrastructure/persistence"
	cvrepo "github.com/intivai/backend/internal/cv/infrastructure/persistence"
	iamapp "github.com/intivai/backend/internal/iam/application"
	"github.com/intivai/backend/internal/iam/infrastructure/auth"
	iamrepo "github.com/intivai/backend/internal/iam/infrastructure/persistence"
	ivapp "github.com/intivai/backend/internal/interview/application"
	ivdomain "github.com/intivai/backend/internal/interview/domain"
	ivrepo "github.com/intivai/backend/internal/interview/infrastructure/persistence"
	jobrepo "github.com/intivai/backend/internal/job/infrastructure/persistence"
	"github.com/intivai/backend/internal/llm"
	scrrepo "github.com/intivai/backend/internal/screening/infrastructure/persistence"
	"github.com/intivai/backend/pkg/db"
	"github.com/intivai/backend/pkg/queue"
	"github.com/intivai/backend/pkg/storage"
	"github.com/rs/zerolog"
	"gorm.io/gorm"
)

// slowStreamLLM — deterministic chunks with delay; aborts when ctx cancels.
type slowStreamLLM struct{}

func (slowStreamLLM) Chat(ctx context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
	return &llm.ChatResponse{Content: "slow response"}, nil
}

func TestTicketInterviewIDRejectsMissingAndMalformedClaims(t *testing.T) {
	cases := []*iamapp.Claims{
		{},
		{Extra: iamapp.TokenExtra{}},
		{Extra: iamapp.TokenExtra{InterviewID: "not-a-uuid"}},
	}
	for _, claims := range cases {
		if id, ok := ticketInterviewID(claims); ok || id != uuid.Nil {
			t.Fatalf("invalid claims accepted: id=%s ok=%v", id, ok)
		}
	}

	want := uuid.New()
	got, ok := ticketInterviewID(&iamapp.Claims{Extra: iamapp.TokenExtra{InterviewID: want.String()}})
	if !ok || got != want {
		t.Fatalf("valid claims rejected: id=%s ok=%v", got, ok)
	}
}
func (slowStreamLLM) ChatStream(ctx context.Context, req llm.ChatRequest) (<-chan string, error) {
	ch := make(chan string)
	go func() {
		defer close(ch)
		for _, tok := range []string{"tok1", "tok2", "tok3", "tok4"} {
			select {
			case <-ctx.Done():
				return
			case <-time.After(100 * time.Millisecond):
				ch <- tok
			}
		}
	}()
	return ch, nil
}
func (slowStreamLLM) StructuredOutput(ctx context.Context, req llm.StructuredRequest) (any, error) {
	return nil, errors.New("unused")
}
func (slowStreamLLM) Embed(ctx context.Context, text string) ([]float32, error) {
	return nil, errors.New("unused")
}
func (slowStreamLLM) CountTokens(text string) int { return 0 }

// seedChatOrg creates org + active job + passed application + structured
// candidate. Returns pool, service, orgID, appID.
func seedChatOrg(t *testing.T) (*gorm.DB, *ivapp.InterviewService, string, string) {
	return seedChatOrgFull(t, nil)
}

// seedChatOrgFull — seedChatOrg with an optional org QA-limit reader factory
// wired into the service (mirrors production main.go wiring via iamrepo; the
// factory receives the pool because the repo needs it).
func seedChatOrgFull(t *testing.T, qaLimitFactory func(pool *gorm.DB) ivapp.OrgQALimitReader) (*gorm.DB, *ivapp.InterviewService, string, string) {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	pool, err := db.NewPool(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	orgID := "cx" + uuid.NewString()[:8]
	orgUUID := uuid.New()
	appID, jobID, candID := uuid.New(), uuid.New(), uuid.New()

	err = db.RunInTx(ctx, pool, orgUUID.String(), func(tctx context.Context) error {
		tx, _ := db.TxFrom(tctx)
		for _, q := range []struct {
			sql  string
			args []any
		}{
			{`INSERT INTO orgs (id, name, slug) VALUES ($1,$2,$3)`, []any{orgUUID, "t", orgID}},
			{`INSERT INTO jobs (id, org_id, title, description, status, created_at) VALUES ($1,$2,$3,$4,'active',NOW())`, []any{jobID, orgUUID, "Go Engineer", "Go backend work"}},
			{`INSERT INTO candidates (id, org_id, name, email, status, created_at) VALUES ($1,$2,$3,$4,'extracted',NOW())`, []any{candID, orgUUID, "Jane", "j@x.io"}},
			{`INSERT INTO applications (id, org_id, candidate_id, job_id, status, cv_score, passed_screening, created_at) VALUES ($1,$2,$3,$4,'passed',80,true,NOW())`, []any{appID, orgUUID, candID, jobID}},
		} {
			if err := tx.Exec(q.sql, q.args...).Error; err != nil {
				return err
			}
		}
		candRepo := cvrepo.NewPostgresCandidateRepo(pool)
		c, err := candRepo.GetByID(tctx, candID)
		if err != nil {
			return err
		}
		c.CVStructured = []byte(`{"skills":["Go"],"experience_years":5,"education":"Master","certifications":[],"summary":"Go engineer"}`)
		return candRepo.Update(tctx, c)
	})
	if err != nil {
		t.Fatal(err)
	}

	minio, err := storage.New(os.Getenv("TEST_MINIO_ENDPOINT"), os.Getenv("TEST_MINIO_ACCESS"), os.Getenv("TEST_MINIO_SECRET"), "intivai", false)
	if err != nil || minio == nil {
		t.Skip("TEST_MINIO_* not set")
	}
	var qaLimit ivapp.OrgQALimitReader
	if qaLimitFactory != nil {
		qaLimit = qaLimitFactory(pool)
	}
	svc := ivapp.NewInterviewService(pool,
		ivrepo.NewPostgresInterviewRepo(pool), ivrepo.NewPostgresTokenRepo(pool), ivrepo.NewPostgresQuestionBank(pool),
		scrrepo.NewPostgresApplicationRepo(pool), cvrepo.NewPostgresCandidateRepo(pool), jobrepo.NewPostgresJobRepo(pool),
		jobrepo.NewPostgresCandidateContextRepo(pool),
		ctxrepo.NewPostgresContextRepo(pool), minio, auth.NewJWTProvider("test-secret-for-chat-flow"), ivdomain.SystemClock(), nil, qaLimit, zerolog.Nop())
	return pool, svc, orgUUID.String(), appID.String()
}

// createInterviewAndTicket — helper: interview + valid ws ticket.
func createInterviewAndTicket(t *testing.T, svc *ivapp.InterviewService, orgID, appID string) (interviewID, ticket string) {
	t.Helper()
	created, err := svc.CreateInterview(context.Background(), actorWith(uuid.MustParse(orgID), "admin"), ivapp.CreateInterviewCommand{ApplicationID: uuid.MustParse(appID), QuestionCount: 3})
	if err != nil {
		t.Fatalf("create interview: %v", err)
	}
	if err := svc.GiveConsent(context.Background(), created.InterviewID, created.Token); err != nil {
		t.Fatalf("consent: %v", err)
	}
	tk, err := svc.IssueTicket(context.Background(), ivapp.IssueTicketCommand{InterviewID: created.InterviewID, InvitationToken: created.Token})
	if err != nil {
		t.Fatalf("issue ticket: %v", err)
	}
	return created.InterviewID.String(), tk.Ticket
}

// chatApp — fiber app with the real chat route for a given service + llm.
func chatApp(svc *ivapp.InterviewService, llmClient llm.Provider, origins []string) *fiber.App {
	app := fiber.New()
	handler := NewChatHandler(svc, llmClient, auth.NewJWTProvider("test-secret-for-chat-flow"), zerolog.Nop())
	app.Get("/candidate/interviews/:id/chat", handler.RequireTicket, handler.Chat(origins))
	return app
}

func dialChatWS(t *testing.T, addr, interviewID, ticket, origin string) (*websocket.Conn, int) {
	t.Helper()
	dialer := websocket.Dialer{HandshakeTimeout: 3 * time.Second}
	headers := map[string][]string{"Authorization": {"Bearer " + ticket}}
	if origin != "" {
		headers["Origin"] = []string{origin}
	}
	conn, resp, err := dialer.Dial("ws://"+addr+"/candidate/interviews/"+interviewID+"/chat", headers)
	if err != nil {
		code := 0
		if resp != nil {
			code = resp.StatusCode
		}
		return nil, code
	}
	return conn, 0
}

func readFrame(t *testing.T, conn *websocket.Conn, timeout time.Duration) (map[string]any, bool) {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(timeout))
	_, raw, err := conn.ReadMessage()
	if err != nil {
		return nil, false
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("bad frame: %s", raw)
	}
	return m, true
}

// RED 1: interrupt must stop the stream mid-response and still advance to the
// next question — no further tokens after "Interrupted.".
func TestChatInterruptStopsStream(t *testing.T) {
	_, svc, orgID, appID := seedChatOrg(t)
	ivID, ticket := createInterviewAndTicket(t, svc, orgID, appID)

	app := chatApp(svc, slowStreamLLM{}, nil)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = app.Listener(ln) }()
	defer func() { _ = app.Shutdown() }()

	conn, code := dialChatWS(t, ln.Addr().String(), ivID, ticket, "")
	if conn == nil {
		t.Fatalf("dial failed: %d", code)
	}
	defer func() { _ = conn.Close() }()

	if _, ok := readFrame(t, conn, 5*time.Second); !ok {
		t.Fatal("no start frame")
	}
	if _, ok := readFrame(t, conn, 5*time.Second); !ok {
		t.Fatal("no question frame")
	}

	if err := conn.WriteJSON(map[string]any{"type": "answer", "content": "my answer", "idx": 1}); err != nil {
		t.Fatal(err)
	}
	if m, ok := readFrame(t, conn, 5*time.Second); !ok || m["type"] != "token" {
		t.Fatalf("expected first token, got %v", m)
	}

	if err := conn.WriteJSON(map[string]string{"type": "interrupt"}); err != nil {
		t.Fatal(err)
	}
	gotInterrupted, gotQuestion := false, false
	for {
		m, ok := readFrame(t, conn, 3*time.Second)
		if !ok {
			break
		}
		if m["type"] == "response" && m["content"] == "Interrupted." {
			gotInterrupted = true
			continue
		}
		switch m["type"] {
		case "token", "response":
			t.Fatalf("%s after interrupt: %v", m["type"], m)
		case "error":
			t.Fatalf("error frame: %v", m)
		case "question":
			gotQuestion = true
		}
		if gotInterrupted && gotQuestion {
			break
		}
	}
	if !gotInterrupted || !gotQuestion {
		t.Fatalf("interrupted=%v question=%v", gotInterrupted, gotQuestion)
	}
}

// RED 2: second concurrent connection to the same interview is rejected and
// the first connection stays functional.
func TestChatSecondConnectionRejected(t *testing.T) {
	_, svc, orgID, appID := seedChatOrg(t)
	created, err := svc.CreateInterview(context.Background(), actorWith(uuid.MustParse(orgID), "admin"), ivapp.CreateInterviewCommand{ApplicationID: uuid.MustParse(appID), QuestionCount: 3})
	if err != nil {
		t.Fatalf("create interview: %v", err)
	}
	if err := svc.GiveConsent(context.Background(), created.InterviewID, created.Token); err != nil {
		t.Fatalf("consent: %v", err)
	}
	tk, err := svc.IssueTicket(context.Background(), ivapp.IssueTicketCommand{InterviewID: created.InterviewID, InvitationToken: created.Token})
	if err != nil {
		t.Fatalf("issue ticket: %v", err)
	}
	ivID := created.InterviewID.String()
	ticket := tk.Ticket

	app := chatApp(svc, slowStreamLLM{}, nil)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = app.Listener(ln) }()
	defer func() { _ = app.Shutdown() }()

	first, code := dialChatWS(t, ln.Addr().String(), ivID, ticket, "")
	if first == nil {
		t.Fatalf("first dial failed: %d", code)
	}
	defer func() { _ = first.Close() }()
	if _, ok := readFrame(t, first, 5*time.Second); !ok {
		t.Fatal("no start")
	}
	if _, ok := readFrame(t, first, 5*time.Second); !ok {
		t.Fatal("no question")
	}

	// A second connection with a DIFFERENT ticket (different session_id) MUST be rejected.
	ticket2, err := svc.IssueTicket(context.Background(), ivapp.IssueTicketCommand{InterviewID: created.InterviewID, InvitationToken: created.Token})
	if err != nil {
		t.Fatalf("issue second ticket: %v", err)
	}

	second, code := dialChatWS(t, ln.Addr().String(), ivID, ticket2.Ticket, "")
	if second == nil {
		t.Fatalf("second dial failed: %d", code)
	}
	defer func() { _ = second.Close() }()
	if m, ok := readFrame(t, second, 5*time.Second); !ok || m["type"] != "error" {
		t.Fatalf("second connection not rejected: %v", m)
	}
	if err := first.WriteJSON(map[string]string{"type": "ping"}); err != nil {
		t.Fatal(err)
	}
	if m, ok := readFrame(t, first, 5*time.Second); !ok || m["type"] != "pong" {
		t.Fatalf("first conn broken after second rejected: %v", m)
	}
}

// RED 3: resume with a mismatched session_id must be rejected.
func TestChatResumeSessionMismatch(t *testing.T) {
	_, svc, orgID, appID := seedChatOrg(t)
	ivID, ticket := createInterviewAndTicket(t, svc, orgID, appID)

	app := chatApp(svc, slowStreamLLM{}, nil)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = app.Listener(ln) }()
	defer func() { _ = app.Shutdown() }()

	conn, code := dialChatWS(t, ln.Addr().String(), ivID, ticket, "")
	if conn == nil {
		t.Fatalf("dial failed: %d", code)
	}
	defer func() { _ = conn.Close() }()
	readFrame(t, conn, 5*time.Second)
	readFrame(t, conn, 5*time.Second)

	if err := conn.WriteJSON(map[string]string{"type": "resume", "session_id": "wrong-session"}); err != nil {
		t.Fatal(err)
	}
	if m, ok := readFrame(t, conn, 5*time.Second); !ok || m["type"] != "error" {
		t.Fatalf("session mismatch not rejected: %v", m)
	}
}

// errorStreamLLM — ChatStream always fails.
type errorStreamLLM struct{}

func (errorStreamLLM) Chat(ctx context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
	return nil, errors.New("upstream down")
}
func (errorStreamLLM) ChatStream(ctx context.Context, req llm.ChatRequest) (<-chan string, error) {
	return nil, errors.New("upstream down")
}
func (errorStreamLLM) StructuredOutput(ctx context.Context, req llm.StructuredRequest) (any, error) {
	return nil, errors.New("unused")
}
func (errorStreamLLM) Embed(ctx context.Context, text string) ([]float32, error) {
	return nil, errors.New("unused")
}
func (errorStreamLLM) CountTokens(text string) int { return 0 }

// RED: LLM failure must not strand the interview — error frame + next
// question still dispatched (answer already recorded).
func TestChatLLMErrorStillAdvances(t *testing.T) {
	_, svc, orgID, appID := seedChatOrg(t)
	ivID, ticket := createInterviewAndTicket(t, svc, orgID, appID)

	app := chatApp(svc, errorStreamLLM{}, nil)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = app.Listener(ln) }()
	defer func() { _ = app.Shutdown() }()

	conn, code := dialChatWS(t, ln.Addr().String(), ivID, ticket, "")
	if conn == nil {
		t.Fatalf("dial failed: %d", code)
	}
	defer func() { _ = conn.Close() }()
	readFrame(t, conn, 5*time.Second)
	readFrame(t, conn, 5*time.Second)

	if err := conn.WriteJSON(map[string]any{"type": "answer", "content": "my answer", "idx": 1}); err != nil {
		t.Fatal(err)
	}
	gotError, gotQuestion := false, false
	for {
		m, ok := readFrame(t, conn, 5*time.Second)
		if !ok {
			break
		}
		switch m["type"] {
		case "error":
			gotError = true
		case "question":
			gotQuestion = true
		}
		if gotError && gotQuestion {
			break
		}
	}
	if !gotError || !gotQuestion {
		t.Fatalf("error=%v question=%v", gotError, gotQuestion)
	}
}

// qaMockLLM — Chat (used by the grounded-answer path) echoes a marker from the
// pinned context so the test can prove the answer was built ONLY from context,
// not from outside knowledge. ChatStream returns a normal interview answer.
type qaMockLLM struct {
	lastQAUser string
}

func (m *qaMockLLM) Chat(ctx context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
	// Record the user message (carries the pinned context) for assertions.
	for _, msg := range req.Messages {
		if msg.Role == "user" {
			m.lastQAUser = msg.Content
		}
	}
	return &llm.ChatResponse{Content: "Based on the role context: this position is fully remote with a Go/Postgres stack."}, nil
}

func (m *qaMockLLM) ChatStream(ctx context.Context, req llm.ChatRequest) (<-chan string, error) {
	ch := make(chan string)
	go func() {
		defer close(ch)
		for _, t := range []string{"mock", " ", "answer"} {
			ch <- t
		}
	}()
	return ch, nil
}
func (m *qaMockLLM) StructuredOutput(ctx context.Context, req llm.StructuredRequest) (any, error) {
	return map[string]any{
		"per_question":   []any{map[string]any{"question_idx": 1, "category": "technical", "score": 80.0, "rationale": "ok", "strengths": []any{}, "weaknesses": []any{}}},
		"strengths":      []any{},
		"weaknesses":     []any{},
		"recommendation": "proceed",
	}, nil
}
func (m *qaMockLLM) Embed(ctx context.Context, text string) ([]float32, error) {
	return nil, errors.New("unused")
}
func (m *qaMockLLM) CountTokens(text string) int { return 0 }

// TestCandidateQuestionAnsweredFromContext — RED/GREEN for B4: a candidate_question
// must be answered STRICTLY from the pinned contexts (job candidate context +
// org company context captured at connect), NEVER advance the interview question
// or touch turn state, and the Q&A pair must be retrievable via GetCandidateQA.
func TestCandidateQuestionAnsweredFromContext(t *testing.T) {
	pool, svc, orgID, appID := seedChatOrg(t)
	ctx := context.Background()
	orgUUID := uuid.MustParse(orgID)

	// Seed a company context (versioned, stored) so LoadQAContext has material.
	minio, err := storage.New(os.Getenv("TEST_MINIO_ENDPOINT"), os.Getenv("TEST_MINIO_ACCESS"), os.Getenv("TEST_MINIO_SECRET"), "intivai", false)
	if err != nil || minio == nil {
		t.Skip("TEST_MINIO_* not set")
	}
	queueClient := queue.NewClient(os.Getenv("TEST_REDIS_ADDR"))
	defer func() { _ = queueClient.Close() }()
	ctxSvc := ctxapp.NewContextService(pool, ctxrepo.NewPostgresContextRepo(pool), minio, queueClient, zerolog.Nop())
	_, err = ctxSvc.UploadContext(ctx, actorWith(orgUUID, "admin"), "text", []byte("Our company builds HR tech. Benefits include unlimited PTO and learning budget."))
	if err != nil {
		t.Fatalf("upload company context: %v", err)
	}

	ivID, ticket := createInterviewAndTicket(t, svc, orgID, appID)

	app := chatApp(svc, &qaMockLLM{}, nil)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = app.Listener(ln) }()
	defer func() { _ = app.Shutdown() }()

	conn, code := dialChatWS(t, ln.Addr().String(), ivID, ticket, "")
	if conn == nil {
		t.Fatalf("dial failed: %d", code)
	}
	defer func() { _ = conn.Close() }()

	// Drain interview.start + first question.
	if _, ok := readFrame(t, conn, 5*time.Second); !ok {
		t.Fatal("no start frame")
	}
	if m, ok := readFrame(t, conn, 5*time.Second); !ok || m["type"] != ivdomain.MsgQuestion {
		t.Fatalf("expected question, got %v", m)
	}

	// MIDPOINT: no QA pair should exist before the candidate asks.
	mid, midErr := svc.GetCandidateQA(ctx, orgUUID.String(), uuid.MustParse(ivID))
	if midErr != nil {
		t.Fatalf("mid GetCandidateQA: %v", midErr)
	}
	t.Logf("pairs before candidate question: %d", len(mid))

	// Send a candidate question.
	if err := conn.WriteJSON(map[string]any{"type": "candidate_question", "content": "What benefits does the role offer?"}); err != nil {
		t.Fatal(err)
	}

	// Expect a qa_answer frame (grounded, NOT refused, NOT a question advance).
	ans, ok := readFrame(t, conn, 10*time.Second)
	if !ok {
		t.Fatal("no qa_answer frame")
	}
	if ans["type"] != ivdomain.MsgQaAnswer {
		t.Fatalf("expected qa_answer, got %v", ans)
	}
	if ans["refused"] == true {
		t.Fatalf("grounded answer refused: %v", ans)
	}
	if ans["question"] != "What benefits does the role offer?" {
		t.Fatalf("qa_answer echoed wrong question: %v", ans)
	}
	if ans["answer"] == nil || ans["answer"].(string) == "" {
		t.Fatalf("qa_answer empty: %v", ans)
	}

	// The pair must be retrievable via GetCandidateQA (recruiter-visible log).
	pairs, err := svc.GetCandidateQA(ctx, orgUUID.String(), uuid.MustParse(ivID))
	if err != nil {
		t.Fatalf("GetCandidateQA: %v", err)
	}
	for i, p := range pairs {
		t.Logf("pair[%d]: question=%q answer=%q", i, p.Question, p.Answer)
	}
	if len(pairs) != 1 {
		t.Fatalf("recorded pairs = %d, want 1", len(pairs))
	}
	if pairs[0].Question != "What benefits does the role offer?" {
		t.Fatalf("recorded question = %q", pairs[0].Question)
	}
}

// qaLimitAdapter mirrors production main.go wiring: IAM org repo behind
// ivapp.OrgQALimitReader.
type qaLimitAdapter struct {
	repo *iamrepo.PostgresIAMRepo
}

func (a qaLimitAdapter) CandidateQALimit(ctx context.Context, orgID uuid.UUID) (int, error) {
	org, err := a.repo.GetOrg(ctx, orgID)
	if err != nil {
		return 0, err
	}
	if org.CandidateQALimit == nil {
		return 0, errors.New("candidate_qa_limit unset")
	}
	return *org.CandidateQALimit, nil
}

// qaCountingLLM counts Chat invocations (the grounded-answer path) so tests can
// prove refusals happen BEFORE the LLM is spent. Chat calls are serialized by
// the handler's busy guard; the counter still guards with a mutex because the
// QA path runs off the read loop.
type qaCountingLLM struct {
	qaMockLLM
	mu    sync.Mutex
	chats int
}

func (m *qaCountingLLM) Chat(ctx context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
	m.mu.Lock()
	m.chats++
	m.mu.Unlock()
	return m.qaMockLLM.Chat(ctx, req)
}

func (m *qaCountingLLM) chatCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.chats
}

// blockingChatLLM — Chat parks until released, so a test can hold a QA answer
// in flight and probe concurrent behavior.
type blockingChatLLM struct {
	qaMockLLM
	entered chan struct{}
	release chan struct{}
}

func newBlockingChatLLM() *blockingChatLLM {
	return &blockingChatLLM{entered: make(chan struct{}), release: make(chan struct{})}
}

func (m *blockingChatLLM) Chat(ctx context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
	select {
	case <-m.entered:
	default:
		close(m.entered)
	}
	select {
	case <-m.release:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	return m.qaMockLLM.Chat(ctx, req)
}

// I4/I5: the org-configured cap must gate BOTH the pre-check and recording at
// the same boundary — with limit=1 the second question is refused BEFORE the
// LLM runs, nothing extra is recorded, and the candidate gets the refusal
// frame instead of a real answer that would never be persisted.
func TestCandidateQuestionRefusedAtOrgCap(t *testing.T) {
	pool, svc, orgID, appID := seedChatOrgFull(t, func(p *gorm.DB) ivapp.OrgQALimitReader {
		return qaLimitAdapter{repo: iamrepo.NewPostgresIAMRepo(p)}
	})
	ctx := context.Background()
	if err := db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
		tx, _ := db.TxFrom(tctx)
		return tx.Exec(`UPDATE orgs SET candidate_qa_limit = 1 WHERE id = $1`, orgID).Error
	}); err != nil {
		t.Fatalf("set org cap: %v", err)
	}

	ivID, ticket := createInterviewAndTicket(t, svc, orgID, appID)
	mock := &qaCountingLLM{}
	app := chatApp(svc, mock, nil)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = app.Listener(ln) }()
	defer func() { _ = app.Shutdown() }()

	conn, code := dialChatWS(t, ln.Addr().String(), ivID, ticket, "")
	if conn == nil {
		t.Fatalf("dial failed: %d", code)
	}
	defer func() { _ = conn.Close() }()
	if _, ok := readFrame(t, conn, 5*time.Second); !ok {
		t.Fatal("no start frame")
	}
	if _, ok := readFrame(t, conn, 5*time.Second); !ok {
		t.Fatal("no question frame")
	}

	// Q1 fits under the cap: answered and recorded.
	if err := conn.WriteJSON(map[string]any{"type": "candidate_question", "content": "First question?"}); err != nil {
		t.Fatal(err)
	}
	first, ok := readFrame(t, conn, 10*time.Second)
	if !ok || first["type"] != ivdomain.MsgQaAnswer || first["refused"] == true {
		t.Fatalf("first qa_answer should be a real answer, got %v", first)
	}
	pairs, err := svc.GetCandidateQA(ctx, orgID, uuid.MustParse(ivID))
	if err != nil || len(pairs) != 1 {
		t.Fatalf("pairs after first ask = %d (%v), want 1", len(pairs), err)
	}

	// Q2 hits the cap: refused BEFORE the LLM, nothing recorded.
	if err := conn.WriteJSON(map[string]any{"type": "candidate_question", "content": "Second question?"}); err != nil {
		t.Fatal(err)
	}
	second, ok := readFrame(t, conn, 10*time.Second)
	if !ok || second["type"] != ivdomain.MsgQaAnswer {
		t.Fatalf("expected qa_answer refusal, got %v", second)
	}
	if second["refused"] != true {
		t.Fatalf("cap-exhausted question was answered, not refused: %v", second)
	}
	if n := mock.chatCount(); n != 1 {
		t.Fatalf("llm Chat calls = %d, want 1 (refusal must precede the LLM)", n)
	}
	pairs, err = svc.GetCandidateQA(ctx, orgID, uuid.MustParse(ivID))
	if err != nil || len(pairs) != 1 {
		t.Fatalf("pairs after capped ask = %d (%v), want 1", len(pairs), err)
	}
}

// I7: an expired interview must not keep answering candidate questions on a
// live socket — the gate fires before the LLM spend and no pair is recorded.
func TestCandidateQuestionRefusedWhenExpired(t *testing.T) {
	pool, svc, orgID, appID := seedChatOrg(t)
	ivID, ticket := createInterviewAndTicket(t, svc, orgID, appID)

	mock := &qaCountingLLM{}
	app := chatApp(svc, mock, nil)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = app.Listener(ln) }()
	defer func() { _ = app.Shutdown() }()

	conn, code := dialChatWS(t, ln.Addr().String(), ivID, ticket, "")
	if conn == nil {
		t.Fatalf("dial failed: %d", code)
	}
	defer func() { _ = conn.Close() }()
	if _, ok := readFrame(t, conn, 5*time.Second); !ok {
		t.Fatal("no start frame")
	}
	if _, ok := readFrame(t, conn, 5*time.Second); !ok {
		t.Fatal("no question frame")
	}

	// Expire mid-session, directly at the storage layer (clock-driven expiry).
	// Tenant tables are RLS FORCED — the update must run in a tenant tx.
	if err := db.RunInTx(context.Background(), pool, orgID, func(tctx context.Context) error {
		tx, _ := db.TxFrom(tctx)
		return tx.Exec(`UPDATE interviews SET expires_at = NOW() - interval '1 hour' WHERE id = $1`, ivID).Error
	}); err != nil {
		t.Fatalf("expire interview: %v", err)
	}

	if err := conn.WriteJSON(map[string]any{"type": "candidate_question", "content": "Any questions about me?"}); err != nil {
		t.Fatal(err)
	}
	frame, ok := readFrame(t, conn, 10*time.Second)
	if !ok || frame["type"] != ivdomain.MsgQaAnswer {
		t.Fatalf("expected qa_answer frame, got %v", frame)
	}
	if frame["refused"] != true {
		t.Fatalf("expired-interview question was answered, not refused: %v", frame)
	}
	if n := mock.chatCount(); n != 0 {
		t.Fatalf("llm Chat calls = %d, want 0 on expired interview", n)
	}
	pairs, err := svc.GetCandidateQA(context.Background(), orgID, uuid.MustParse(ivID))
	if err != nil || len(pairs) != 0 {
		t.Fatalf("pairs on expired interview = %d (%v), want 0", len(pairs), err)
	}
}

// I3: a candidate question arriving while another QA answer is in flight must
// be refused immediately — never queued into double LLM spend or cap races,
// and never blocking the read loop.
func TestCandidateQuestionBusyRefusedWhileInFlight(t *testing.T) {
	_, svc, orgID, appID := seedChatOrg(t)
	ivID, ticket := createInterviewAndTicket(t, svc, orgID, appID)

	mock := newBlockingChatLLM()
	app := chatApp(svc, mock, nil)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = app.Listener(ln) }()
	defer func() { _ = app.Shutdown() }()

	conn, code := dialChatWS(t, ln.Addr().String(), ivID, ticket, "")
	if conn == nil {
		t.Fatalf("dial failed: %d", code)
	}
	defer func() { _ = conn.Close() }()
	if _, ok := readFrame(t, conn, 5*time.Second); !ok {
		t.Fatal("no start frame")
	}
	if _, ok := readFrame(t, conn, 5*time.Second); !ok {
		t.Fatal("no question frame")
	}

	if err := conn.WriteJSON(map[string]any{"type": "candidate_question", "content": "Slow one?"}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-mock.entered:
	case <-time.After(5 * time.Second):
		t.Fatal("QA handler never reached the LLM")
	}

	if err := conn.WriteJSON(map[string]any{"type": "candidate_question", "content": "While busy?"}); err != nil {
		t.Fatal(err)
	}
	busyFrame, ok := readFrame(t, conn, 3*time.Second)
	if !ok || busyFrame["type"] != ivdomain.MsgQaAnswer {
		t.Fatalf("expected immediate busy qa_answer, got %v", busyFrame)
	}
	if busyFrame["refused"] != true {
		t.Fatalf("concurrent QA was processed instead of refused: %v", busyFrame)
	}

	close(mock.release)
	ansFrame, ok := readFrame(t, conn, 10*time.Second)
	if !ok || ansFrame["type"] != ivdomain.MsgQaAnswer || ansFrame["refused"] == true {
		t.Fatalf("in-flight QA answer lost: %v", ansFrame)
	}
	pairs, err := svc.GetCandidateQA(context.Background(), orgID, uuid.MustParse(ivID))
	if err != nil || len(pairs) != 1 {
		t.Fatalf("pairs = %d (%v), want 1 (busy ask records nothing)", len(pairs), err)
	}
}

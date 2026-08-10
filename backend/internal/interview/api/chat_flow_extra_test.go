package api

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"os"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	ctxrepo "github.com/intivai/backend/internal/context/infrastructure/persistence"
	cvrepo "github.com/intivai/backend/internal/cv/infrastructure/persistence"
	"github.com/intivai/backend/internal/iam/infrastructure/auth"
	ivapp "github.com/intivai/backend/internal/interview/application"
	ivdomain "github.com/intivai/backend/internal/interview/domain"
	ivrepo "github.com/intivai/backend/internal/interview/infrastructure/persistence"
	jobrepo "github.com/intivai/backend/internal/job/infrastructure/persistence"
	"github.com/intivai/backend/internal/llm"
	scrrepo "github.com/intivai/backend/internal/screening/infrastructure/persistence"
	"github.com/intivai/backend/pkg/db"
	"github.com/intivai/backend/pkg/storage"
	"github.com/rs/zerolog"
	"gorm.io/gorm"
)

// slowStreamLLM — deterministic chunks with delay; aborts when ctx cancels.
type slowStreamLLM struct{}

func (slowStreamLLM) Chat(ctx context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
	return &llm.ChatResponse{Content: "slow response"}, nil
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
	svc := ivapp.NewInterviewService(pool,
		ivrepo.NewPostgresInterviewRepo(pool), ivrepo.NewPostgresTokenRepo(pool), ivrepo.NewPostgresQuestionBank(pool),
		scrrepo.NewPostgresApplicationRepo(pool), cvrepo.NewPostgresCandidateRepo(pool), jobrepo.NewPostgresJobRepo(pool),
		ctxrepo.NewPostgresContextRepo(pool), minio, auth.NewJWTProvider("test-secret-for-chat-flow"), ivdomain.SystemClock())
	return pool, svc, orgUUID.String(), appID.String()
}

// createInterviewAndTicket — helper: interview + valid ws ticket.
func createInterviewAndTicket(t *testing.T, svc *ivapp.InterviewService, orgID, appID string) (interviewID, ticket string) {
	t.Helper()
	created, err := svc.CreateInterview(context.Background(), actorWith(uuid.MustParse(orgID), "admin"), ivapp.CreateInterviewCommand{ApplicationID: uuid.MustParse(appID), QuestionCount: 3})
	if err != nil {
		t.Fatalf("create interview: %v", err)
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
	ivID, ticket := createInterviewAndTicket(t, svc, orgID, appID)

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

	second, code := dialChatWS(t, ln.Addr().String(), ivID, ticket, "")
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

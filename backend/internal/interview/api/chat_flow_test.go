package api

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"net"
	"os"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gorilla/websocket"
	ctxrepo "github.com/intivai/backend/internal/context/infrastructure/persistence"
	cvrepo "github.com/intivai/backend/internal/cv/infrastructure/persistence"
	iamapp "github.com/intivai/backend/internal/iam/application"
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
)

type streamMockLLM struct{}

func (streamMockLLM) Chat(ctx context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
	return &llm.ChatResponse{Content: "mock response"}, nil
}
func (streamMockLLM) ChatStream(ctx context.Context, req llm.ChatRequest) (<-chan string, error) {
	ch := make(chan string)
	go func() {
		defer close(ch)
		for _, t := range []string{"mock", " ", "answer"} {
			ch <- t
		}
	}()
	return ch, nil
}
func (streamMockLLM) StructuredOutput(ctx context.Context, req llm.StructuredRequest) (any, error) {
	return nil, errors.New("unused")
}
func (streamMockLLM) Embed(ctx context.Context, text string) ([]float32, error) {
	return nil, errors.New("unused")
}
func (streamMockLLM) CountTokens(text string) int { return 0 }

// Full chat flow: create interview (passed application) → issue ticket →
// ws connect with ticket → ping/pong → answer → streamed tokens + next
// question. Requires live Postgres + Redis + MinIO.
func TestChatFlowEndToEnd(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	pool, err := db.NewPool(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	orgID := "chat" + time.Now().Format("150405") // unique-ish slug suffix
	orgUUID := uuid.New()
	appID, jobID, candID := uuid.New(), uuid.New(), uuid.New()

	// Seed: org + passed application with structured candidate + job.
	err = db.RunInTx(ctx, pool, orgUUID.String(), func(tctx context.Context) error {
		tx, _ := db.TxFrom(tctx)
		queries := []struct {
			sql  string
			args []any
		}{
			{`INSERT INTO orgs (id, name, slug) VALUES ($1,$2,$3)`, []any{orgUUID, "t", orgID}},
			{`INSERT INTO jobs (id, org_id, title, description, status, created_at) VALUES ($1,$2,$3,$4,'active',NOW())`, []any{jobID, orgUUID, "Go Engineer", "Go backend work"}},
			{`INSERT INTO candidates (id, org_id, name, email, status, created_at) VALUES ($1,$2,$3,$4,'extracted',NOW())`, []any{candID, orgUUID, "Jane", "j@x.io"}},
			{`INSERT INTO applications (id, org_id, candidate_id, job_id, status, cv_score, passed_screening, created_at) VALUES ($1,$2,$3,$4,'passed',80,true,NOW())`, []any{appID, orgUUID, candID, jobID}},
		}
		for _, q := range queries {
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

	// MinIO optional for context prompt — construct store; prompt compose
	// tolerates missing contexts.
	minio, err := storage.New(os.Getenv("TEST_MINIO_ENDPOINT"), os.Getenv("TEST_MINIO_ACCESS"), os.Getenv("TEST_MINIO_SECRET"), "intivai", false)
	if err != nil || minio == nil {
		t.Skip("TEST_MINIO_* not set")
	}

	jwt := auth.NewJWTProvider("test-secret-for-chat-flow")
	svc := ivapp.NewInterviewService(pool,
		ivrepo.NewPostgresInterviewRepo(pool), ivrepo.NewPostgresTokenRepo(pool), ivrepo.NewPostgresQuestionBank(pool),
		scrrepo.NewPostgresApplicationRepo(pool), cvrepo.NewPostgresCandidateRepo(pool), jobrepo.NewPostgresJobRepo(pool),
		ctxrepo.NewPostgresContextRepo(pool), minio, jwt, ivdomain.SystemClock())

	// 1. Create interview (recruiter role).
	created, err := svc.CreateInterview(ctx, actorWith(orgUUID, "admin"), ivapp.CreateInterviewCommand{ApplicationID: appID, QuestionCount: 3})
	if err != nil {
		t.Fatalf("create interview: %v", err)
	}

	// 2. Issue ticket with the invitation token.
	ticket, err := svc.IssueTicket(ctx, ivapp.IssueTicketCommand{InterviewID: created.InterviewID, InvitationToken: created.Token})
	if err != nil {
		t.Fatalf("issue ticket: %v", err)
	}

	// 3. Serve the chat route in-process.
	app := fiber.New()
	handler := NewChatHandler(svc, streamMockLLM{}, jwt, zerolog.Nop())
	app.Get("/candidate/interviews/:id", handler.RequireTicket, handler.Chat())

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = app.Listener(ln) }()
	defer func() { _ = app.Shutdown() }()

	dialer := websocket.Dialer{HandshakeTimeout: 3 * time.Second}
	headers := map[string][]string{"Authorization": {"Bearer " + ticket.Ticket}}
	conn, resp, err := dialer.Dial("ws://"+ln.Addr().String()+"/candidate/interviews/"+created.InterviewID.String(), headers)
	if err != nil {
		t.Fatalf("ws dial: %v (%d)", err, resp.StatusCode)
	}
	defer func() { _ = conn.Close() }()

	// 4. Expect interview.start then first question.
	read := func() map[string]any {
		_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		_, raw, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		var m map[string]any
		if err := json.Unmarshal(raw, &m); err != nil {
			t.Fatalf("bad frame: %s", raw)
		}
		return m
	}
	start := read()
	if start["type"] != ivdomain.MsgStart || start["total_questions"].(float64) != 3 {
		t.Fatalf("start frame: %v", start)
	}
	q1 := read()
	if q1["type"] != ivdomain.MsgQuestion {
		t.Fatalf("expected question, got %v", q1)
	}

	// 5. ping → pong.
	if err := conn.WriteJSON(map[string]string{"type": "ping"}); err != nil {
		t.Fatal(err)
	}
	pong := read()
	if pong["type"] != "pong" {
		t.Fatalf("pong frame: %v", pong)
	}

	// 6. Answer → token stream → response → next question.
	if err := conn.WriteJSON(map[string]any{"type": "answer", "content": "I built Go services", "idx": 1}); err != nil {
		t.Fatal(err)
	}
	tokens := []string{}
	for {
		m := read()
		switch m["type"] {
		case ivdomain.MsgToken:
			tokens = append(tokens, m["content"].(string))
		case ivdomain.MsgResponse:
			if len(tokens) == 0 {
				t.Fatalf("response without tokens: %v", m)
			}
			if m["content"].(string) != "mock answer" {
				t.Fatalf("response content: %v", m)
			}
			goto nextQuestion
		default:
			t.Fatalf("unexpected frame %v", m)
		}
	}
nextQuestion:
	q2 := read()
	if q2["type"] != ivdomain.MsgQuestion {
		t.Fatalf("expected next question, got %v", q2)
	}

	// 7. Wrong ticket rejected (401, no upgrade).
	dialer2 := websocket.Dialer{HandshakeTimeout: 2 * time.Second}
	_, _, err = dialer2.Dial("ws://"+ln.Addr().String()+"/candidate/interviews/"+created.InterviewID.String(), map[string][]string{"Authorization": {"Bearer not-a-ticket"}})
	if err == nil {
		t.Fatal("invalid ticket accepted")
	}
}

func actorWith(orgID uuid.UUID, role string) iamapp.AuthContext {
	return iamapp.AuthContext{OrgID: orgID, Role: role}
}

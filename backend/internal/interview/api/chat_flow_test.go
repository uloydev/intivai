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

// lastStreamRequest records the most recent ChatStream request so tests can
// assert the context window passed to the LLM.
var lastStreamRequest llm.ChatRequest

func (streamMockLLM) Chat(ctx context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
	return &llm.ChatResponse{Content: "mock response"}, nil
}
func (streamMockLLM) ChatStream(ctx context.Context, req llm.ChatRequest) (<-chan string, error) {
	lastStreamRequest = req
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
	return map[string]any{
		"per_question": []any{
			map[string]any{"question_idx": 1, "category": "technical", "score": 80.0, "rationale": "ok", "strengths": []any{}, "weaknesses": []any{}},
			map[string]any{"question_idx": 2, "category": "communication", "score": 70.0, "rationale": "ok", "strengths": []any{}, "weaknesses": []any{}},
			map[string]any{"question_idx": 3, "category": "problem_solving", "score": 60.0, "rationale": "ok", "strengths": []any{}, "weaknesses": []any{}},
		},
		"strengths":      []any{},
		"weaknesses":     []any{},
		"recommendation": "proceed",
	}, nil
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
		ctxrepo.NewPostgresContextRepo(pool), minio, jwt, ivdomain.SystemClock(), nil)

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
	app.Get("/candidate/interviews/:id", handler.RequireTicket, handler.Chat(nil))

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

	// 6b. Context window: second stream request must carry the history pair
	// (assistant question 1 + user answer 1) after the system prompt.
	if len(lastStreamRequest.Messages) != 3 {
		t.Fatalf("stream request messages = %d, want 3 (system + q1 + a1)", len(lastStreamRequest.Messages))
	}
	if lastStreamRequest.Messages[0].Role != "system" ||
		lastStreamRequest.Messages[1].Role != "assistant" ||
		lastStreamRequest.Messages[2].Role != "user" ||
		lastStreamRequest.Messages[2].Content != "I built Go services" {
		t.Fatalf("history window wrong: %+v", lastStreamRequest.Messages)
	}

	// 7. Answer the second question; window grows to 5 messages.
	if err := conn.WriteJSON(map[string]any{"type": "answer", "content": "I ship production Go", "idx": 2}); err != nil {
		t.Fatal(err)
	}
	for {
		m := read()
		if m["type"] == ivdomain.MsgResponse {
			break
		}
	}
	if len(lastStreamRequest.Messages) != 5 {
		t.Fatalf("stream request messages = %d, want 5 (system + 2 pairs)", len(lastStreamRequest.Messages))
	}

	// 7b. Detailed answers (no probe) drive to completion; the final frame is
	// the evaluation with REAL scores (not the old empty map).
	for i := 0; i < 6; i++ {
		if err := conn.WriteJSON(map[string]any{"type": "answer", "content": "I have deep experience with distributed systems and production microservices at scale"}); err != nil {
			t.Fatal(err)
		}
		for {
			m := read()
			switch m["type"] {
			case ivdomain.MsgToken, ivdomain.MsgQuestion:
				continue
			case ivdomain.MsgResponse:
				goto answered
			case ivdomain.MsgEvaluation:
				if m["status"] != "complete" {
					t.Fatalf("evaluation status = %v, want complete", m["status"])
				}
				scores, ok := m["scores"].(map[string]any)
				if !ok || len(scores) == 0 {
					t.Fatalf("evaluation scores empty: %v", m)
				}
				if m["overall"].(float64) <= 0 {
					t.Fatalf("evaluation overall = %v, want > 0", m["overall"])
				}
				goto evaluated
			default:
				t.Fatalf("unexpected frame %v", m)
			}
		}
	answered:
	}
	t.Fatal("interview never completed (no evaluation frame)")
evaluated:

	// 7c. Evaluation persisted to the interviews row.
	var evalJSON []byte
	err = db.RunInTx(ctx, pool, orgUUID.String(), func(tctx context.Context) error {
		tx, _ := db.TxFrom(tctx)
		return tx.Raw(`SELECT evaluation FROM interviews WHERE id = $1`, created.InterviewID).Row().Scan(&evalJSON)
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(evalJSON) == 0 {
		t.Fatal("evaluation not persisted")
	}

	// 8. Wrong ticket rejected (401, no upgrade).
	dialer2 := websocket.Dialer{HandshakeTimeout: 2 * time.Second}
	_, _, err = dialer2.Dial("ws://"+ln.Addr().String()+"/candidate/interviews/"+created.InterviewID.String(), map[string][]string{"Authorization": {"Bearer not-a-ticket"}})
	if err == nil {
		t.Fatal("invalid ticket accepted")
	}
}

func actorWith(orgID uuid.UUID, role string) iamapp.AuthContext {
	return iamapp.AuthContext{OrgID: orgID, Role: role}
}

package api

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"

	ctxrepo "github.com/intivai/backend/internal/context/infrastructure/persistence"
	cvrepo "github.com/intivai/backend/internal/cv/infrastructure/persistence"
	"github.com/intivai/backend/internal/iam/infrastructure/auth"
	ivapp "github.com/intivai/backend/internal/interview/application"
	ivdomain "github.com/intivai/backend/internal/interview/domain"
	ivrepo "github.com/intivai/backend/internal/interview/infrastructure/persistence"
	jobrepo "github.com/intivai/backend/internal/job/infrastructure/persistence"
	scrrepo "github.com/intivai/backend/internal/screening/infrastructure/persistence"
	"github.com/intivai/backend/pkg/db"
	"github.com/intivai/backend/pkg/storage"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type spanMemExporter struct {
	mu    sync.Mutex
	spans []sdktrace.ReadOnlySpan
}

func (m *spanMemExporter) ExportSpans(_ context.Context, spans []sdktrace.ReadOnlySpan) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.spans = append(m.spans, spans...)
	return nil
}

func (m *spanMemExporter) Shutdown(context.Context) error { return nil }

// all returns a snapshot under the lock — spans export from the stream
// goroutine concurrently with assertions.
func (m *spanMemExporter) all() []sdktrace.ReadOnlySpan {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]sdktrace.ReadOnlySpan, len(m.spans))
	copy(out, m.spans)
	return out
}

// The WS answer path must emit interview.* turn spans so one candidate answer
// reads as a waterfall: process → stream (plan batch E). Env-gated: full
// protocol harness against real service.
func TestChatFlowEmitsTurnSpans(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	minio, err := storage.New(os.Getenv("TEST_MINIO_ENDPOINT"), os.Getenv("TEST_MINIO_ACCESS"), os.Getenv("TEST_MINIO_SECRET"), "intivai", false)
	if err != nil || minio == nil {
		t.Skip("TEST_MINIO_* not set")
	}

	exp := &spanMemExporter{}
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exp))
	prevTP := otel.GetTracerProvider()
	otel.SetTracerProvider(tp)
	t.Cleanup(func() { otel.SetTracerProvider(prevTP) })

	pool, err := db.NewPool(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	orgUUID := uuid.New()
	appID, jobID, candID := uuid.New(), uuid.New(), uuid.New()

	if err := db.RunInTx(ctx, pool, orgUUID.String(), func(tctx context.Context) error {
		tx, _ := db.TxFrom(tctx)
		for _, q := range []struct {
			sql  string
			args []any
		}{
			{`INSERT INTO orgs (id, name, slug) VALUES ($1,$2,$3)`, []any{orgUUID, "t", "spans" + orgUUID.String()[:8]}},
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
	}); err != nil {
		t.Fatal(err)
	}

	jwt := auth.NewJWTProvider("test-secret-for-chat-flow")
	svc := ivapp.NewInterviewService(pool,
		ivrepo.NewPostgresInterviewRepo(pool), ivrepo.NewPostgresTokenRepo(pool), ivrepo.NewPostgresQuestionBank(pool),
		scrrepo.NewPostgresApplicationRepo(pool), cvrepo.NewPostgresCandidateRepo(pool), jobrepo.NewPostgresJobRepo(pool),
		jobrepo.NewPostgresCandidateContextRepo(pool),
		ctxrepo.NewPostgresContextRepo(pool), minio, jwt, ivdomain.SystemClock(), nil, nil, zerolog.Nop())

	created, err := svc.CreateInterview(ctx, actorWith(orgUUID, "admin"), ivapp.CreateInterviewCommand{ApplicationID: appID, QuestionCount: 3})
	if err != nil {
		t.Fatalf("create interview: %v", err)
	}
	if err := svc.GiveConsent(ctx, created.InterviewID, created.Token); err != nil {
		t.Fatalf("consent: %v", err)
	}
	ticket, err := svc.IssueTicket(ctx, ivapp.IssueTicketCommand{InterviewID: created.InterviewID, InvitationToken: created.Token})
	if err != nil {
		t.Fatalf("ticket: %v", err)
	}

	app := fiber.New()
	handler := NewChatHandler(svc, streamMockLLM{}, jwt, zerolog.Nop())
	app.Get("/candidate/interviews/:id", handler.RequireTicket, handler.Chat(nil))
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = app.Listener(ln) }()
	t.Cleanup(func() { _ = app.Shutdown() })

	dialer := websocket.Dialer{HandshakeTimeout: 3 * time.Second}
	headers := map[string][]string{"Authorization": {"Bearer " + ticket.Ticket}}
	conn, resp, err := dialer.Dial("ws://"+ln.Addr().String()+"/candidate/interviews/"+created.InterviewID.String(), headers)
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		t.Fatalf("ws dial: %v (%d)", err, resp.StatusCode)
	}
	defer func() { _ = conn.Close() }()

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
	if m := read(); m["type"] != ivdomain.MsgStart {
		t.Fatalf("want start, got %v", m["type"])
	}
	if m := read(); m["type"] != ivdomain.MsgQuestion {
		t.Fatalf("want question, got %v", m["type"])
	}

	if err := conn.WriteJSON(map[string]any{"type": "answer", "content": "I built Go services", "idx": 1}); err != nil {
		t.Fatal(err)
	}
	for {
		m := read()
		if m["type"] == ivdomain.MsgQuestion {
			break
		}
		if m["type"] != ivdomain.MsgToken && m["type"] != ivdomain.MsgResponse {
			t.Fatalf("unexpected frame %v", m)
		}
	}

	found := map[string]bool{
		"interview.answer.process": false,
		"interview.turn.stream":    false,
	}
	for _, s := range exp.all() {
		if _, ok := found[s.Name()]; ok {
			found[s.Name()] = true
			if !s.Parent().IsValid() && s.Name() != "interview.answer.process" {
				t.Fatalf("span %q must nest under the answer process span (orphan trace = ctx dropped)", s.Name())
			}
		}
	}
	for name, ok := range found {
		if !ok {
			names := make([]string, 0)
			for _, s := range exp.all() {
				names = append(names, s.Name())
			}
			t.Fatalf("turn span %q missing; exported: %v", name, names)
		}
	}
}

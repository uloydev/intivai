package application

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	evalllm "github.com/intivai/backend/internal/evaluation/infrastructure/llm"
	ivdomain "github.com/intivai/backend/internal/interview/domain"
	ivrepo "github.com/intivai/backend/internal/interview/infrastructure/persistence"
	"github.com/intivai/backend/internal/llm"
	notifapp "github.com/intivai/backend/internal/notification/application"
	"github.com/intivai/backend/pkg/db"
	"github.com/rs/zerolog"
)

type mockEvalProvider struct{}

func (mockEvalProvider) Chat(ctx context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
	return nil, nil
}
func (mockEvalProvider) ChatStream(ctx context.Context, req llm.ChatRequest) (<-chan string, error) {
	return nil, nil
}
func (mockEvalProvider) StructuredOutput(ctx context.Context, req llm.StructuredRequest) (any, error) {
	return map[string]any{
		"per_question": []any{
			map[string]any{"question_idx": 1, "category": "technical", "score": 90.0, "rationale": "r", "strengths": []any{}, "weaknesses": []any{}},
		},
		"strengths": []any{}, "weaknesses": []any{}, "recommendation": "proceed",
	}, nil
}
func (mockEvalProvider) Embed(ctx context.Context, text string) ([]float32, error) {
	return nil, nil
}
func (mockEvalProvider) CountTokens(text string) int { return 0 }

// Worker happy path + idempotency: evaluation persisted on first run; replay
// skips (no double LLM, evaluation untouched).
func TestEvaluationWorkerHappyPathAndIdempotent(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	pool, err := db.NewPool(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	orgID := uuid.NewString()
	ivID, appID, jobID, candID := uuid.New(), uuid.New(), uuid.New(), uuid.New()

	err = db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
		tx, _ := db.TxFrom(tctx)
		for _, q := range []struct {
			sql  string
			args []any
		}{
			{`INSERT INTO orgs (id, name, slug) VALUES ($1,$2,$3)`, []any{orgID, "t", "ew" + orgID[:8]}},
			{`INSERT INTO jobs (id, org_id, title, description, status, created_at) VALUES ($1,$2,$3,$4,'active',NOW())`, []any{jobID, orgID, "Go", "Go"}},
			{`INSERT INTO candidates (id, org_id, name, email, status, created_at) VALUES ($1,$2,$3,$4,'extracted',NOW())`, []any{candID, orgID, "Jane", "j@x.io"}},
			{`INSERT INTO applications (id, org_id, candidate_id, job_id, status, cv_score, passed_screening, created_at) VALUES ($1,$2,$3,$4,'passed',80,true,NOW())`, []any{appID, orgID, candID, jobID}},
		} {
			if err := tx.Exec(q.sql, q.args...).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(struct {
		Questions []ivdomain.Question `json:"questions"`
		Answers   []ivdomain.Answer   `json:"answers"`
	}{
		Questions: []ivdomain.Question{{Idx: 1, Content: "q1", Category: "technical"}},
		Answers:   []ivdomain.Answer{{Idx: 1, Content: "an answer with enough words to pass probing", AnsweredAt: time.Now()}},
	})
	err = db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
		tx, _ := db.TxFrom(tctx)
		return tx.Exec(`INSERT INTO interviews (id, application_id, type, status, transcript, last_question_idx, context_version, created_at)
			VALUES ($1, $2, 'chat', 'completed', $3, 1, 0, NOW())`, ivID, appID, raw).Error
	})
	if err != nil {
		t.Fatal(err)
	}

	repo := ivrepo.NewPostgresInterviewRepo(pool)
	worker := NewEvaluationWorker(pool, repo, evalllm.NewEvaluator(mockEvalProvider{}), nil, "http://localhost:5173", zerolog.Nop())

	payload, _ := json.Marshal(EvaluatePayload{OrgID: orgID, InterviewID: ivID.String()})
	task := asynq.NewTask(TaskEvaluateInterview, payload)
	if err := worker.handle(ctx, task); err != nil {
		t.Fatalf("worker: %v", err)
	}

	var evalJSON []byte
	err = db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
		tx, _ := db.TxFrom(tctx)
		return tx.Raw(`SELECT evaluation FROM interviews WHERE id = $1`, ivID).Row().Scan(&evalJSON)
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(evalJSON) == 0 {
		t.Fatal("evaluation not persisted")
	}

	// Replay: must skip (idempotent), evaluation unchanged.
	if err := worker.handle(ctx, task); !errors.Is(err, asynq.SkipRetry) {
		t.Fatalf("worker replay err = %v, want asynq.SkipRetry", err)
	}
	var evalAfter []byte
	_ = db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
		tx, _ := db.TxFrom(tctx)
		return tx.Raw(`SELECT evaluation FROM interviews WHERE id = $1`, ivID).Row().Scan(&evalAfter)
	})
	if string(evalJSON) != string(evalAfter) {
		t.Fatal("replay changed the evaluation")
	}

	// Atomic guard: a direct second save (simulated inline-vs-worker race)
	// must be rejected — first writer wins, value untouched.
	err = db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
		return repo.SaveEvaluation(tctx, ivID, []byte(`{"overall_score": 1}`))
	})
	if !errors.Is(err, ivdomain.ErrEvaluationExists) {
		t.Fatalf("second save err = %v, want ErrEvaluationExists", err)
	}
	var evalFinal []byte
	_ = db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
		tx, _ := db.TxFrom(tctx)
		return tx.Raw(`SELECT evaluation FROM interviews WHERE id = $1`, ivID).Row().Scan(&evalFinal)
	})
	if string(evalFinal) != string(evalJSON) {
		t.Fatal("atomic guard overwritten the evaluation")
	}
}

type capturedEmail struct {
	Task    string
	Payload notifapp.SendEmailPayload
}

type capturingEnqueuer struct{ sent []capturedEmail }

func (c *capturingEnqueuer) Enqueue(_ context.Context, jobType string, payload any, _ ...asynq.Option) (*asynq.TaskInfo, error) {
	p, ok := payload.(notifapp.SendEmailPayload)
	if !ok {
		return nil, errors.New("unexpected enqueue payload type")
	}
	c.sent = append(c.sent, capturedEmail{Task: jobType, Payload: p})
	return &asynq.TaskInfo{}, nil
}

// Scorecard notification: recipients must be resolved through a tenant tx —
// users is FORCED RLS, so a raw pool query sees zero rows and recruiters are
// silently never emailed. Only the owning org's admin/recruiters get mail.
func TestEvaluationWorkerScorecardEmailsResolveOrgUsers(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	pool, err := db.NewPool(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	orgID := uuid.NewString()
	otherOrgID := uuid.NewString()
	adminID, recruiterID, foreignUserID := uuid.New(), uuid.New(), uuid.New()
	ivID, appID, jobID, candID := uuid.New(), uuid.New(), uuid.New(), uuid.New()

	err = db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
		tx, _ := db.TxFrom(tctx)
		for _, q := range []struct {
			sql  string
			args []any
		}{
			{`INSERT INTO orgs (id, name, slug) VALUES ($1,$2,$3)`, []any{orgID, "t", "ew2" + orgID[:8]}},
			{`INSERT INTO users (id, org_id, email, role) VALUES ($1,$2,$3,'admin')`, []any{adminID, orgID, "admin-" + orgID[:8] + "@x.io"}},
			{`INSERT INTO users (id, org_id, email, role) VALUES ($1,$2,$3,'recruiter')`, []any{recruiterID, orgID, "rec-" + orgID[:8] + "@x.io"}},
			{`INSERT INTO jobs (id, org_id, title, description, status, created_at) VALUES ($1,$2,$3,$4,'active',NOW())`, []any{jobID, orgID, "Go", "Go"}},
			{`INSERT INTO candidates (id, org_id, name, email, status, created_at) VALUES ($1,$2,$3,$4,'extracted',NOW())`, []any{candID, orgID, "Jane", "j@x.io"}},
			{`INSERT INTO applications (id, org_id, candidate_id, job_id, status, cv_score, passed_screening, created_at) VALUES ($1,$2,$3,$4,'passed',80,true,NOW())`, []any{appID, orgID, candID, jobID}},
		} {
			if err := tx.Exec(q.sql, q.args...).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	// Foreign org + its admin: separate tenant tx (RLS INSERT policy).
	err = db.RunInTx(ctx, pool, otherOrgID, func(tctx context.Context) error {
		tx, _ := db.TxFrom(tctx)
		for _, q := range []struct {
			sql  string
			args []any
		}{
			{`INSERT INTO orgs (id, name, slug) VALUES ($1,$2,$3)`, []any{otherOrgID, "o", "ew2" + otherOrgID[:8]}},
			{`INSERT INTO users (id, org_id, email, role) VALUES ($1,$2,$3,'admin')`, []any{foreignUserID, otherOrgID, "foreign-" + otherOrgID[:8] + "@x.io"}},
		} {
			if err := tx.Exec(q.sql, q.args...).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(struct {
		Questions []ivdomain.Question `json:"questions"`
		Answers   []ivdomain.Answer   `json:"answers"`
	}{
		Questions: []ivdomain.Question{{Idx: 1, Content: "q1", Category: "technical"}},
		Answers:   []ivdomain.Answer{{Idx: 1, Content: "an answer with enough words to pass probing", AnsweredAt: time.Now()}},
	})
	err = db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
		tx, _ := db.TxFrom(tctx)
		return tx.Exec(`INSERT INTO interviews (id, application_id, type, status, transcript, last_question_idx, context_version, created_at)
			VALUES ($1, $2, 'chat', 'completed', $3, 1, 0, NOW())`, ivID, appID, raw).Error
	})
	if err != nil {
		t.Fatal(err)
	}

	enq := &capturingEnqueuer{}
	repo := ivrepo.NewPostgresInterviewRepo(pool)
	worker := NewEvaluationWorker(pool, repo, evalllm.NewEvaluator(mockEvalProvider{}), enq, "http://localhost:5173", zerolog.Nop())

	payload, _ := json.Marshal(EvaluatePayload{OrgID: orgID, InterviewID: ivID.String()})
	task := asynq.NewTask(TaskEvaluateInterview, payload)
	if err := worker.handle(ctx, task); err != nil {
		t.Fatalf("worker: %v", err)
	}

	want := map[string]bool{
		"admin-" + orgID[:8] + "@x.io": false,
		"rec-" + orgID[:8] + "@x.io":   false,
	}
	for _, c := range enq.sent {
		if c.Task != notifapp.TaskSendEmail {
			t.Fatalf("task = %s, want %s", c.Task, notifapp.TaskSendEmail)
		}
		if c.Payload.Type != notifapp.EmailTypeScorecard {
			t.Fatalf("email type = %s, want %s", c.Payload.Type, notifapp.EmailTypeScorecard)
		}
		if _, ok := want[c.Payload.To]; !ok {
			t.Fatalf("unexpected recipient %q (cross-org leak)", c.Payload.To)
		}
		want[c.Payload.To] = true
	}
	for to, got := range want {
		if !got {
			t.Fatalf("scorecard email to %q never sent: got %d sends", to, len(enq.sent))
		}
	}
}

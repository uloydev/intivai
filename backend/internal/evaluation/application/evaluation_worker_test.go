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
	"github.com/intivai/backend/pkg/db"
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
	worker := NewEvaluationWorker(pool, repo, evalllm.NewEvaluator(mockEvalProvider{}))

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
}

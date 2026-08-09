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
	cvdomain "github.com/intivai/backend/internal/cv/domain"
	cvrepo "github.com/intivai/backend/internal/cv/infrastructure/persistence"
	"github.com/intivai/backend/internal/llm"
	scrrepo "github.com/intivai/backend/internal/screening/infrastructure/persistence"
	shareddomain "github.com/intivai/backend/internal/shared/domain"
	"github.com/intivai/backend/pkg/db"
	"github.com/intivai/backend/pkg/queue"
	"github.com/rs/zerolog"
)

type failingLLM struct{}

func (failingLLM) Chat(ctx context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
	return nil, errors.New("unused")
}
func (failingLLM) ChatStream(ctx context.Context, req llm.ChatRequest) (<-chan string, error) {
	return nil, errors.New("unused")
}
func (failingLLM) StructuredOutput(ctx context.Context, req llm.StructuredRequest) (any, error) {
	return nil, errors.New("llm 500")
}
func (failingLLM) Embed(ctx context.Context, text string) ([]float32, error) {
	return nil, errors.New("unused")
}
func (failingLLM) CountTokens(text string) int { return 0 }

// Integration test — live Postgres + Redis. Guards the failure path:
// LLM failure → candidate failed_extract with error_message, task SkipRetry.
func TestExtractWorkerFailurePath(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	redisAddr := os.Getenv("TEST_REDIS_ADDR")
	if redisAddr == "" {
		t.Skip("TEST_REDIS_ADDR not set")
	}
	pool, err := db.NewPool(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	orgID := uuid.NewString()
	candID := uuid.New()

	err = db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
		tx, ok := db.TxFrom(tctx)
		if !ok {
			return db.ErrNoTx
		}
		if err := tx.Exec(`INSERT INTO orgs (id, name, slug) VALUES ($1,$2,$3)`, orgID, "t", "f"+orgID[:8]).Error; err != nil {
			return err
		}
		c := &cvdomain.Candidate{
			Entity: shareddomain.Entity{ID: candID, CreatedAt: time.Now().UTC()},
			OrgID:  uuid.MustParse(orgID), Name: "Jane", Status: "parsed", CVRawText: "resume",
		}
		return cvrepo.NewPostgresCandidateRepo(pool).Create(tctx, c)
	})
	if err != nil {
		t.Fatal(err)
	}

	worker := NewExtractWorker(pool, cvrepo.NewPostgresCandidateRepo(pool), scrrepo.NewPostgresApplicationRepo(pool), nil, failingLLM{}, queue.NewClient(redisAddr), zerolog.Nop())
	payload, _ := json.Marshal(ExtractCVPayload{OrgID: orgID, CandidateID: candID.String()})
	err = worker.handle(ctx, asynq.NewTask(TaskExtractCV, payload))
	if err == nil || !errors.Is(err, asynq.SkipRetry) {
		t.Fatalf("expected SkipRetry on LLM failure, got %v", err)
	}

	var cand *cvdomain.Candidate
	err = db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
		cand, err = cvrepo.NewPostgresCandidateRepo(pool).GetByID(tctx, candID)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if cand.Status != cvdomain.StatusFailedExtract {
		t.Fatalf("status = %s, want failed_extract", cand.Status)
	}
	if cand.ErrorMessage == "" {
		t.Fatal("error_message not persisted")
	}
}

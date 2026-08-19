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
	jobrepo "github.com/intivai/backend/internal/job/infrastructure/persistence"
	"github.com/intivai/backend/internal/llm"
	scrdomain "github.com/intivai/backend/internal/screening/domain"
	scrrepo "github.com/intivai/backend/internal/screening/infrastructure/persistence"
	shareddomain "github.com/intivai/backend/internal/shared/domain"
	"github.com/intivai/backend/pkg/db"
	"github.com/intivai/backend/pkg/queue"
	"github.com/rs/zerolog"
)

type stubLLM struct {
	calls int
}

func (s *stubLLM) Chat(ctx context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
	return nil, errors.New("unused")
}
func (s *stubLLM) ChatStream(ctx context.Context, req llm.ChatRequest) (<-chan string, error) {
	return nil, errors.New("unused")
}
func (s *stubLLM) StructuredOutput(ctx context.Context, req llm.StructuredRequest) (any, error) {
	s.calls++
	return &ResumeData{Skills: []string{"Go"}, ExperienceYears: 3, Education: "Bachelor", Summary: "Go dev"}, nil
}
func (s *stubLLM) Embed(ctx context.Context, text string) ([]float32, error) {
	return nil, errors.New("unused")
}
func (s *stubLLM) CountTokens(text string) int { return 0 }

// Integration test — live Postgres + Redis. Guards extract idempotency:
// a retry after successful extraction must NOT re-run the LLM (call count
// stays 1) or duplicate applications.
func TestExtractWorkerIdempotentRetry(t *testing.T) {
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

	llmStub := &stubLLM{}
	worker := NewExtractWorker(pool, cvrepo.NewPostgresCandidateRepo(pool), scrrepo.NewPostgresApplicationRepo(pool), jobrepo.NewPostgresJobRepo(pool), llmStub, queue.NewClient(redisAddr), "http://localhost:5173", zerolog.Nop())

	seedCandidate := func(status string) {
		t.Helper()
		err = db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
			tx, ok := db.TxFrom(tctx)
			if !ok {
				return db.ErrNoTx
			}
			if err := tx.Exec(`INSERT INTO orgs (id, name, slug) VALUES ($1,$2,$3)`, orgID, "t", "e"+orgID[:8]).Error; err != nil {
				return err
			}
			c := &cvdomain.Candidate{
				Entity: shareddomain.Entity{ID: candID, CreatedAt: time.Now().UTC()},
				OrgID:  uuid.MustParse(orgID), Name: "Jane", Status: status, CVRawText: "Go engineer resume",
			}
			return cvrepo.NewPostgresCandidateRepo(pool).Create(tctx, c)
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	seedCandidate("parsed")

	payload, _ := json.Marshal(ExtractCVPayload{OrgID: orgID, CandidateID: candID.String()})
	if err := worker.handle(ctx, asynq.NewTask(TaskExtractCV, payload)); err != nil {
		t.Fatalf("first extract: %v", err)
	}
	if llmStub.calls != 1 {
		t.Fatalf("llm calls = %d, want 1", llmStub.calls)
	}

	// Retry after commit (crash window): must skip LLM, keep status extracted.
	if err := worker.handle(ctx, asynq.NewTask(TaskExtractCV, payload)); err != nil && !errors.Is(err, asynq.SkipRetry) {
		t.Fatalf("retry extract: %v", err)
	}
	if llmStub.calls != 1 {
		t.Fatalf("llm calls after retry = %d, want 1 (idempotent)", llmStub.calls)
	}

	var apps []*scrdomain.Application
	err = db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
		apps, err = scrrepo.NewPostgresApplicationRepo(pool).List(tctx, uuid.MustParse(orgID), uuid.Nil)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(apps) != 0 {
		t.Fatalf("applications created without active jobs: %d", len(apps))
	}

	var cand *cvdomain.Candidate
	err = db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
		cand, err = cvrepo.NewPostgresCandidateRepo(pool).GetByID(tctx, candID)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if cand.Status != cvdomain.StatusPendingReview || len(cand.CVStructured) == 0 {
		t.Fatalf("candidate state: %s structured=%d", cand.Status, len(cand.CVStructured))
	}
}

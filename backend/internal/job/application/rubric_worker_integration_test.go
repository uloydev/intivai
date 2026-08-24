package application

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	jobdomain "github.com/intivai/backend/internal/job/domain"

	jobrepo "github.com/intivai/backend/internal/job/infrastructure/persistence"
	"github.com/intivai/backend/internal/llm"
	"github.com/intivai/backend/pkg/db"
	"github.com/rs/zerolog"
	"gorm.io/gorm"
)

// rubricStubProvider counts LLM invocations so retry/idempotency semantics
// are observable; it can also mutate the job row mid-call to simulate a
// recruiter edit racing the generation.
type rubricStubProvider struct {
	calls      int
	fail       bool
	duringCall func(ctx context.Context) error
}

type rubricFixtureSchema struct {
	Summary    string            `json:"summary"`
	Dimensions []rubricDimSchema `json:"dimensions"`
}

type rubricDimSchema struct {
	Name             string   `json:"name"`
	Description      string   `json:"description"`
	WeightPercentage int      `json:"weight_percentage"`
	Criteria         []string `json:"criteria"`
}

func (s *rubricStubProvider) Chat(ctx context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
	return nil, errors.New("unused")
}
func (s *rubricStubProvider) ChatStream(ctx context.Context, req llm.ChatRequest) (<-chan string, error) {
	return nil, errors.New("unused")
}
func (s *rubricStubProvider) StructuredOutput(ctx context.Context, req llm.StructuredRequest) (any, error) {
	s.calls++
	if s.duringCall != nil {
		if err := s.duringCall(ctx); err != nil {
			return nil, err
		}
	}
	if s.fail {
		return nil, errors.New("llm upstream down")
	}
	return &rubricFixtureSchema{
		Summary: "Strong Go backend engineer",
		Dimensions: []rubricDimSchema{
			{Name: "Technical Skills", Description: "Go depth", WeightPercentage: 60, Criteria: []string{"5+ years Go"}},
			{Name: "Experience", Description: "Backend scale", WeightPercentage: 40, Criteria: []string{"Distributed systems"}},
		},
	}, nil
}
func (s *rubricStubProvider) Embed(ctx context.Context, text string) ([]float32, error) {
	return nil, errors.New("unused")
}
func (s *rubricStubProvider) CountTokens(text string) int { return 0 }

func newTestRubricWorker(pool *gorm.DB, prov llm.Provider) *RubricWorker {
	client := llm.NewClient(prov, nil, nil, 1)
	return NewRubricWorker(pool, jobrepo.NewPostgresJobRepo(pool), client, zerolog.Nop())
}

func seedRubricOrgJob(t *testing.T, pool *gorm.DB) (orgID string, jobID uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	orgID = uuid.NewString()
	jobID = uuid.New()
	err := db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
		tx, ok := db.TxFrom(tctx)
		if !ok {
			return db.ErrNoTx
		}
		if err := tx.Exec(`INSERT INTO orgs (id, name, slug) VALUES ($1,$2,$3)`, orgID, "t", "r"+orgID[:8]).Error; err != nil {
			return err
		}
		return tx.Exec(
			`INSERT INTO jobs (id, org_id, title, description, status, created_at)
			 VALUES ($1,$2,'Go Engineer','orig-description','active',NOW())`,
			jobID, orgID).Error
	})
	if err != nil {
		t.Fatal(err)
	}
	return orgID, jobID
}

func rubricTask(orgID string, jobID uuid.UUID) *asynq.Task {
	payload, _ := json.Marshal(GenerateRubricPayload{JobID: jobID.String(), OrgID: orgID})
	return asynq.NewTask(TaskGenerateRubric, payload)
}

func loadJobRubric(t *testing.T, pool *gorm.DB, orgID string, jobID uuid.UUID) (*jobdomain.Job, error) {
	t.Helper()
	var job *jobdomain.Job
	err := db.RunInTx(context.Background(), pool, orgID, func(tctx context.Context) error {
		var err error
		job, err = jobrepo.NewPostgresJobRepo(pool).GetByID(tctx, jobID)
		return err
	})
	return job, err
}

// Happy path: the rubric persists, AND a concurrent recruiter edit made while
// the LLM was generating survives (proves the write is column-scoped, not a
// full-row clobber).
func TestRubricWorkerHappyPathPreservesConcurrentEdit(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	pool, err := db.NewPool(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	orgID, jobID := seedRubricOrgJob(t, pool)

	prov := &rubricStubProvider{
		duringCall: func(ctx context.Context) error {
			return db.RunInTx(context.Background(), pool, orgID, func(tctx context.Context) error {
				tx, ok := db.TxFrom(tctx)
				if !ok {
					return db.ErrNoTx
				}
				return tx.Exec(`UPDATE jobs SET description = 'edited-while-llm-ran' WHERE id = $1`, jobID).Error
			})
		},
	}
	worker := newTestRubricWorker(pool, prov)

	if err := worker.handle(context.Background(), rubricTask(orgID, jobID)); err != nil {
		t.Fatalf("handle: %v", err)
	}
	if prov.calls != 1 {
		t.Fatalf("llm calls = %d, want 1", prov.calls)
	}

	job, err := loadJobRubric(t, pool, orgID, jobID)
	if err != nil {
		t.Fatal(err)
	}
	if len(job.Rubric) == 0 || string(job.Rubric) == "null" {
		t.Fatalf("rubric not persisted: %q", string(job.Rubric))
	}
	var parsed rubricFixtureSchema
	if err := json.Unmarshal(job.Rubric, &parsed); err != nil || parsed.Summary == "" || len(parsed.Dimensions) != 2 {
		t.Fatalf("persisted rubric invalid: %q (%v)", string(job.Rubric), err)
	}
	if job.Description != "edited-while-llm-ran" {
		t.Fatalf("concurrent edit clobbered: description = %q — full-row UPDATE suspected", job.Description)
	}
}

// Retry after the rubric already persisted must NOT re-invoke the LLM.
func TestRubricWorkerRetryAfterSuccessSkipsLLM(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	pool, err := db.NewPool(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	orgID, jobID := seedRubricOrgJob(t, pool)
	prov := &rubricStubProvider{}
	worker := newTestRubricWorker(pool, prov)

	if err := worker.handle(context.Background(), rubricTask(orgID, jobID)); err != nil {
		t.Fatalf("first handle: %v", err)
	}
	if err := worker.handle(context.Background(), rubricTask(orgID, jobID)); err != nil && !errors.Is(err, asynq.SkipRetry) {
		t.Fatalf("retry handle: %v", err)
	}
	if prov.calls != 1 {
		t.Fatalf("llm calls after retry = %d, want 1 (result-gated idempotency)", prov.calls)
	}
}

// Retry AFTER A FAILED generation re-invokes the LLM (nothing was persisted,
// so there is nothing to gate on); the eventual success lands exactly once.
func TestRubricWorkerRetryAfterFailureRegenerates(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	pool, err := db.NewPool(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	orgID, jobID := seedRubricOrgJob(t, pool)
	prov := &rubricStubProvider{fail: true}
	worker := newTestRubricWorker(pool, prov)

	if err := worker.handle(context.Background(), rubricTask(orgID, jobID)); err == nil {
		t.Fatal("expected failure path to return an error (asynq retries)")
	}
	prov.fail = false
	if err := worker.handle(context.Background(), rubricTask(orgID, jobID)); err != nil {
		t.Fatalf("retry after failure: %v", err)
	}
	if prov.calls != 2 {
		t.Fatalf("llm calls = %d, want 2 (failed attempt + successful retry)", prov.calls)
	}
	job, err := loadJobRubric(t, pool, orgID, jobID)
	if err != nil || len(job.Rubric) == 0 {
		t.Fatalf("rubric missing after recovery: %q (%v)", string(job.Rubric), err)
	}
}

// Tenant discipline: an org-B task naming an org-A job sees nothing through
// RLS (SkipRetry), never invokes the LLM, and writes no rubric.
func TestRubricWorkerCrossOrgIsolation(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	pool, err := db.NewPool(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	orgA, jobID := seedRubricOrgJob(t, pool)
	orgB := uuid.NewString()
	err = db.RunInTx(context.Background(), pool, orgB, func(tctx context.Context) error {
		tx, ok := db.TxFrom(tctx)
		if !ok {
			return db.ErrNoTx
		}
		return tx.Exec(`INSERT INTO orgs (id, name, slug) VALUES ($1,$2,$3)`, orgB, "b", "b"+orgB[:8]).Error
	})
	if err != nil {
		t.Fatal(err)
	}

	prov := &rubricStubProvider{}
	worker := newTestRubricWorker(pool, prov)
	err = worker.handle(context.Background(), rubricTask(orgB, jobID))
	if err != nil && !errors.Is(err, asynq.SkipRetry) {
		t.Fatalf("cross-org job must be permanent-skip, got %v", err)
	}
	if prov.calls != 0 {
		t.Fatalf("llm invoked across tenant boundary: %d calls", prov.calls)
	}
	jobA, err := loadJobRubric(t, pool, orgA, jobID)
	if err != nil || len(jobA.Rubric) > 0 {
		t.Fatalf("cross-org write leaked: %q (%v)", string(jobA.Rubric), err)
	}
}

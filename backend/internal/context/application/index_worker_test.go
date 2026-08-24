package application

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	ctxdomain "github.com/intivai/backend/internal/context/domain"
	"github.com/intivai/backend/pkg/db"
	"github.com/rs/zerolog"
	"gorm.io/gorm"
)

type fakeContextRepo struct {
	getErr    error
	got       *ctxdomain.CompanyContext
	getCalled bool
}

func (f *fakeContextRepo) CreateContext(ctx context.Context, cc *ctxdomain.CompanyContext) error {
	return nil
}
func (f *fakeContextRepo) GetContextByID(ctx context.Context, id uuid.UUID) (*ctxdomain.CompanyContext, error) {
	f.getCalled = true
	if f.getErr != nil {
		return nil, f.getErr
	}
	if f.got == nil {
		return nil, ctxdomain.ErrNotFound
	}
	return f.got, nil
}
func (f *fakeContextRepo) GetContextByHash(ctx context.Context, orgID uuid.UUID, hash string) (*ctxdomain.CompanyContext, error) {
	return nil, ctxdomain.ErrNotFound
}
func (f *fakeContextRepo) ListContexts(ctx context.Context, orgID uuid.UUID) ([]*ctxdomain.CompanyContext, error) {
	return nil, nil
}
func (f *fakeContextRepo) DeleteContext(ctx context.Context, orgID, id uuid.UUID) error {
	return nil
}
func (f *fakeContextRepo) SetPrompt(ctx context.Context, p *ctxdomain.TenantPrompt) error { return nil }
func (f *fakeContextRepo) GetLatestPrompt(ctx context.Context, orgID uuid.UUID) (*ctxdomain.TenantPrompt, error) {
	return nil, ctxdomain.ErrNotFound
}

func newIndexWorker(repo *fakeContextRepo) *IndexWorker {
	return NewIndexWorker(nil, repo, nil, nil, zerolog.Nop())
}

func indexTask(orgID string) *asynq.Task {
	payload, _ := json.Marshal(IndexContextPayload{OrgID: orgID, ContextID: uuid.NewString()})
	return asynq.NewTask(TaskIndexContext, payload)
}

func indexTxCtx() context.Context {
	return db.WithTx(context.Background(), &gorm.DB{})
}

// Transient load failure must return a retryable error — SkipRetry here
// silently strands the context unindexed forever (finding D20).
func TestIndexWorkerTransientLoadFailureRetries(t *testing.T) {
	repo := &fakeContextRepo{getErr: errors.New("pgx: pool exhausted")}

	err := newIndexWorker(repo).handle(indexTxCtx(), indexTask(uuid.NewString()))
	if err == nil || errors.Is(err, asynq.SkipRetry) {
		t.Fatalf("transient load failure returned %v, want retryable error", err)
	}
	if !errors.Is(err, repo.getErr) {
		t.Fatalf("error %v does not carry the cause for logs", err)
	}
	if !repo.getCalled {
		t.Fatal("context was never loaded")
	}
}

// A context row that does not exist will never exist: permanent → SkipRetry.
func TestIndexWorkerMissingContextIsPermanent(t *testing.T) {
	repo := &fakeContextRepo{} // returns ErrNotFound

	err := newIndexWorker(repo).handle(indexTxCtx(), indexTask(uuid.NewString()))
	if !errors.Is(err, asynq.SkipRetry) {
		t.Fatalf("missing context is permanent, got %v", err)
	}
}

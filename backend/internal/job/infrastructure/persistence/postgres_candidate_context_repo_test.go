package persistence

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/google/uuid"
	jobdomain "github.com/intivai/backend/internal/job/domain"
	"github.com/intivai/backend/pkg/db"
)

// TestCandidateContextRoundTrip + RLS isolation: covers NULL-free round-trip,
// version bump on each save, and cross-org isolation. Env-gated (needs
// TEST_DATABASE_URL).
func TestCandidateContextRoundTrip(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	pool, err := db.NewPool(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	orgA := uuid.NewString()
	seedOrg(t, pool, orgA, "cca"+orgA[:8])
	jobRepo := NewPostgresJobRepo(pool)
	ccRepo := NewPostgresCandidateContextRepo(pool)

	job, err := jobdomain.NewJob(uuid.MustParse(orgA), "Backend Engineer", "Build APIs.", []string{"Go"}, 3)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.RunInTx(ctx, pool, orgA, func(tctx context.Context) error {
		return jobRepo.Create(tctx, job)
	}); err != nil {
		t.Fatalf("seed job: %v", err)
	}

	cc1, err := jobdomain.NewCandidateContext(uuid.MustParse(orgA), job.ID, "Remote role, $120k-$150k.")
	if err != nil {
		t.Fatal(err)
	}
	if err := db.RunInTx(ctx, pool, orgA, func(tctx context.Context) error {
		return ccRepo.Upsert(tctx, cc1)
	}); err != nil {
		t.Fatalf("upsert v1: %v", err)
	}

	var got *jobdomain.CandidateContext
	err = db.RunInTx(ctx, pool, orgA, func(tctx context.Context) error {
		got, err = ccRepo.GetByJobID(tctx, job.ID)
		return err
	})
	if err != nil {
		t.Fatalf("get v1: %v", err)
	}
	if got.Version != 1 || got.Content != "Remote role, $120k-$150k." {
		t.Fatalf("v1 mismatch: %+v", got)
	}

	// Save again → version must bump to 2.
	cc2, err := jobdomain.NewCandidateContext(uuid.MustParse(orgA), job.ID, "Updated: hybrid, $130k-$160k.")
	if err != nil {
		t.Fatal(err)
	}
	if err := db.RunInTx(ctx, pool, orgA, func(tctx context.Context) error {
		return ccRepo.Upsert(tctx, cc2)
	}); err != nil {
		t.Fatalf("upsert v2: %v", err)
	}
	err = db.RunInTx(ctx, pool, orgA, func(tctx context.Context) error {
		got, err = ccRepo.GetByJobID(tctx, job.ID)
		return err
	})
	if err != nil {
		t.Fatalf("get v2: %v", err)
	}
	if got.Version != 2 {
		t.Fatalf("version after second save = %d, want 2", got.Version)
	}

	// RLS isolation: org B must NOT see org A's context for the same job id.
	orgB := uuid.NewString()
	seedOrg(t, pool, orgB, "ccb"+orgB[:8])
	var bErr error
	_ = db.RunInTx(ctx, pool, orgB, func(tctx context.Context) error {
		_, bErr = ccRepo.GetByJobID(tctx, job.ID)
		return nil
	})
	if bErr == nil {
		t.Fatal("org B must not read org A's candidate context (RLS isolation broken)")
	}
	if !errors.Is(bErr, jobdomain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for cross-org read, got: %v", bErr)
	}
}

// Ensure the postgres repo satisfies the domain contract.
var _ jobdomain.CandidateContextRepository = (*PostgresCandidateContextRepo)(nil)

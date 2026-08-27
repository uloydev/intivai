package persistence

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	jobdomain "github.com/intivai/backend/internal/job/domain"
	"github.com/intivai/backend/pkg/db"
	"gorm.io/gorm"
)

func seedOrg(t *testing.T, pool *gorm.DB, orgID, slug string) {
	t.Helper()
	ctx := context.Background()
	if err := db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
		tx, _ := db.TxFrom(tctx)
		return tx.Exec(`INSERT INTO orgs (id, name, slug) VALUES ($1,$2,$3)`, orgID, "t", slug).Error
	}); err != nil {
		t.Fatal(err)
	}
}

// Round-trip: create with minimal (NULL) fields, update, list, get.
// Guards NULL-scan crashes + jsonb round-trips — the historical bug class.
func TestJobRoundTrip(t *testing.T) {
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
	seedOrg(t, pool, orgID, "jt"+orgID[:8])

	repo := NewPostgresJobRepo(pool)

	// Minimal job: NULL required_skills / min_experience / weights / threshold.
	minimal, err := jobdomain.NewJob(uuid.MustParse(orgID), "Minimal", "Desc", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
		return repo.Create(tctx, minimal)
	}); err != nil {
		t.Fatalf("create minimal: %v", err)
	}

	var got *jobdomain.Job
	err = db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
		got, err = repo.GetByID(tctx, minimal.ID)
		return err
	})
	if err != nil {
		t.Fatalf("get minimal (NULL scans): %v", err)
	}
	if got.Title != "Minimal" || got.RequiredSkills != nil || got.MinExperience != 0 {
		t.Fatalf("minimal round-trip mismatch: %+v", got)
	}

	// Full update: skills + weights + threshold + archived.
	got.RequiredSkills = []string{"Go", "PostgreSQL"}
	got.MinExperience = 3
	got.MinScoreToProceed = float64Ptr(60)
	if err := got.SetScoringWeights(map[string]float64{
		"skills_match": 0.4, "experience_years": 0.2, "semantic_match": 0.2,
		"education": 0.1, "certifications": 0.1,
	}); err != nil {
		t.Fatal(err)
	}
	got.Status = jobdomain.StatusArchived
	if err := db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
		return repo.Update(tctx, got)
	}); err != nil {
		t.Fatalf("update: %v", err)
	}

	var updated *jobdomain.Job
	err = db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
		updated, err = repo.GetByID(tctx, minimal.ID)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.RequiredSkills) != 2 || updated.MinExperience != 3 ||
		updated.MinScoreToProceed == nil || *updated.MinScoreToProceed != 60 ||
		updated.ScoringWeights["skills_match"] != 0.4 || updated.Status != jobdomain.StatusArchived {
		t.Fatalf("updated round-trip mismatch: %+v", updated)
	}

	var list []*jobdomain.Job
	err = db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
		list, err = repo.List(tctx, uuid.MustParse(orgID))
		return err
	})
	if err != nil || len(list) != 1 {
		t.Fatalf("list: %d rows, err %v", len(list), err)
	}
	var active []*jobdomain.Job
	err = db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
		active, err = repo.ListActive(tctx, uuid.MustParse(orgID))
		return err
	})
	if err != nil || len(active) != 0 {
		t.Fatalf("ListActive (archived job must be excluded): %d rows, err %v", len(active), err)
	}
}

func TestPublicJobQueries(t *testing.T) {
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
	slug := "pub-" + orgID[:8]
	seedOrg(t, pool, orgID, slug)

	repo := NewPostgresJobRepo(pool)

	job, err := jobdomain.NewJob(uuid.MustParse(orgID), "Public Role", "Public Desc", []string{"Go", "React"}, 2)
	if err != nil {
		t.Fatal(err)
	}
	job.Location = "Remote"
	job.EmploymentType = "Full-time"
	job.Responsibilities = []string{"Write code", "Review PRs"}
	job.Requirements = []string{"5+ years Go", "PostgreSQL proficiency"}
	job.NiceToHaves = []string{"Docker", "Kubernetes"}
	job.Benefits = []string{"Health insurance", "401k"}
	job.IsPublished = true

	if err := db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
		return repo.Create(tctx, job)
	}); err != nil {
		t.Fatalf("create public job: %v", err)
	}

	// 1. Test ListPublicActive
	publicJobs, err := repo.ListPublicActive(ctx, slug)
	if err != nil {
		t.Fatalf("ListPublicActive: %v", err)
	}
	if len(publicJobs) == 0 {
		t.Fatalf("expected at least 1 public job, got 0")
	}
	if publicJobs[0].Title != "Public Role" || publicJobs[0].OrgSlug != slug {
		t.Fatalf("public job mismatch: %+v", publicJobs[0])
	}

	// 2. Test GetPublicDetail
	detail, err := repo.GetPublicDetail(ctx, job.ID)
	if err != nil {
		t.Fatalf("GetPublicDetail: %v", err)
	}
	if detail.Title != "Public Role" || len(detail.RequiredSkills) != 2 {
		t.Fatalf("public detail mismatch: %+v", detail)
	}
}

func float64Ptr(v float64) *float64 { return &v }

// Finding I8 / D7: Update must report the row version it just stamped
// (RETURNING updated_at) so callers can derive per-transition task IDs.
func TestJobUpdateReturnsBumpedUpdatedAt(t *testing.T) {
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
	seedOrg(t, pool, orgID, "uab"+orgID[:8])

	repo := NewPostgresJobRepo(pool)
	job, err := jobdomain.NewJob(uuid.MustParse(orgID), "Versioned", "Desc", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
		return repo.Create(tctx, job)
	}); err != nil {
		t.Fatalf("create: %v", err)
	}
	if job.UpdatedAt.IsZero() {
		t.Fatal("create left updated_at unset")
	}

	before := job.UpdatedAt
	time.Sleep(2 * time.Millisecond)
	if err := db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
		return repo.Update(tctx, job)
	}); err != nil {
		t.Fatalf("update: %v", err)
	}
	if !job.UpdatedAt.After(before) {
		t.Fatalf("update did not bump updated_at: before=%v after=%v", before, job.UpdatedAt)
	}
}

// Finding I9 / D8: GetByIDForUpdate must hold the row lock — a concurrent
// locked read of the same row blocks until the first tx ends, while the plain
// MVCC read still succeeds. statement_timeout makes the blocking deterministic.
func TestGetByIDForUpdateSerializesConcurrentReaders(t *testing.T) {
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
	seedOrg(t, pool, orgID, "luk"+orgID[:8])

	repo := NewPostgresJobRepo(pool)
	job, err := jobdomain.NewJob(uuid.MustParse(orgID), "Locked", "Desc", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
		return repo.Create(tctx, job)
	}); err != nil {
		t.Fatal(err)
	}

	err = db.RunInTx(ctx, pool, orgID, func(txA context.Context) error {
		if _, err := repo.GetByIDForUpdate(txA, job.ID); err != nil {
			return err
		}
		// Tx B: same pool → separate connection → separate transaction.
		// Plain read: MVCC snapshot, unaffected by the lock.
		if err := db.RunInTx(ctx, pool, orgID, func(txB context.Context) error {
			_, err := repo.GetByID(txB, job.ID)
			return err
		}); err != nil {
			return fmt.Errorf("plain GetByID blocked unexpectedly: %w", err)
		}
		// Locked read: must time out waiting for tx A. The statement timeout
		// ABORTS that throwaway transaction (Postgres semantics), so it runs
		// last and its error is the expectation.
		blockedErr := db.RunInTx(ctx, pool, orgID, func(txB context.Context) error {
			tx, _ := db.TxFrom(txB)
			if err := tx.Exec(`SET LOCAL statement_timeout = '300ms'`).Error; err != nil {
				return err
			}
			_, err := repo.GetByIDForUpdate(txB, job.ID)
			return err
		})
		if blockedErr == nil {
			t.Fatal("FOR UPDATE read did not block on concurrently-locked row")
		}
		if !strings.Contains(blockedErr.Error(), "statement timeout") &&
			!strings.Contains(blockedErr.Error(), "canceling statement") {
			return fmt.Errorf("locked read failed for unexpected reason: %w", blockedErr)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// Finding I14 / D9: the worker's terminal failure state (jobs.question_set_error)
// must survive the repo read path so the GET DTO can surface it. NULL scans to
// "" — a healthy job carries no error.
func TestJobQuestionSetErrorRoundTrip(t *testing.T) {
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
	seedOrg(t, pool, orgID, "qse"+orgID[:8])

	repo := NewPostgresJobRepo(pool)
	job, err := jobdomain.NewJob(uuid.MustParse(orgID), "ErrState", "Desc", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
		return repo.Create(tctx, job)
	}); err != nil {
		t.Fatalf("create: %v", err)
	}

	var got *jobdomain.Job
	err = db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
		got, err = repo.GetByID(tctx, job.ID)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.QuestionSetError != "" {
		t.Fatalf("fresh job carries error state: %q", got.QuestionSetError)
	}

	const msg = "question set invalid: provider returned no questions"
	if err := db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
		tx, _ := db.TxFrom(tctx)
		return tx.Exec(`UPDATE jobs SET question_set_error = $1 WHERE id = $2`, msg, job.ID).Error
	}); err != nil {
		t.Fatal(err)
	}

	err = db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
		got, err = repo.GetByID(tctx, job.ID)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.QuestionSetError != msg {
		t.Fatalf("question_set_error round-trip = %q, want %q", got.QuestionSetError, msg)
	}
}

package persistence

import (
	"context"
	"os"
	"testing"

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
	if err := got.SetScoringWeights(map[string]float64{"skills_match": 0.4}); err != nil {
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

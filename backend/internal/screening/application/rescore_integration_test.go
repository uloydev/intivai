package application

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	cvdomain "github.com/intivai/backend/internal/cv/domain"
	cvrepo "github.com/intivai/backend/internal/cv/infrastructure/persistence"
	"github.com/intivai/backend/internal/iam/application"
	iamdomain "github.com/intivai/backend/internal/iam/domain"
	iamrepo "github.com/intivai/backend/internal/iam/infrastructure/persistence"
	jobdomain "github.com/intivai/backend/internal/job/domain"
	jobrepo "github.com/intivai/backend/internal/job/infrastructure/persistence"
	scrdomain "github.com/intivai/backend/internal/screening/domain"
	scrrepo "github.com/intivai/backend/internal/screening/infrastructure/persistence"
	shareddomain "github.com/intivai/backend/internal/shared/domain"
	"github.com/intivai/backend/pkg/db"
	"github.com/intivai/backend/pkg/queue"
)

// Integration test — requires live Postgres (migrations applied) + Redis.
// Guards the savepoint recovery: a duplicate application create (23505) must
// not poison the transaction, and re-triggering score must work.
// Run: make test-integration-dev
func TestScreeningReScoreAfterDuplicate(t *testing.T) {
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
	jobID := uuid.New()
	candID := uuid.New()

	err = db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
		tx, ok := db.TxFrom(tctx)
		if !ok {
			return db.ErrNoTx
		}
		if err := tx.Exec(`INSERT INTO orgs (id, name, slug) VALUES ($1,$2,$3)`, orgID, "t", "t"+orgID[:8]).Error; err != nil {
			return err
		}
		if err := tx.Exec(`INSERT INTO jobs (id, org_id, title, description, status, created_at) VALUES ($1,$2,$3,$4,'active',NOW())`, jobID, orgID, "J", "d").Error; err != nil {
			return err
		}
		candRepo := cvrepo.NewPostgresCandidateRepo(pool)
		c := &cvdomain.Candidate{
			Entity: shareddomain.Entity{ID: candID, CreatedAt: time.Now().UTC()},
			OrgID:  uuid.MustParse(orgID), Name: "x", Email: "x@x.io", Status: "parsed",
		}
		if err := candRepo.Create(tctx, c); err != nil {
			return err
		}
		c.CVStructured = []byte(`{"skills":["Go"],"experience_years":3,"education":"Bachelor","certifications":[],"summary":"Go dev"}`)
		c.Status = "extracted"
		return candRepo.Update(tctx, c)
	})
	if err != nil {
		t.Fatal(err)
	}

	svc := NewScreeningService(pool, scrrepo.NewPostgresApplicationRepo(pool), cvrepo.NewPostgresCandidateRepo(pool), jobrepo.NewPostgresJobRepo(pool), queue.NewClient(redisAddr))
	actor := application.AuthContext{OrgID: uuid.MustParse(orgID), Role: string(iamdomain.RoleAdmin)}

	if _, err := svc.Create(ctx, actor, CreateScreeningCommand{CandidateID: candID, JobID: jobID}); err != nil {
		t.Fatalf("first create: %v", err)
	}
	// Duplicate create must recover via savepoint and re-trigger, not 500.
	if _, err := svc.Create(ctx, actor, CreateScreeningCommand{CandidateID: candID, JobID: jobID}); err != nil {
		t.Fatalf("re-score create (savepoint recovery): %v", err)
	}

	var apps []*ApplicationResult
	err = db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
		apps, err = svc.List(tctx, actor, uuid.Nil)
		return err
	})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(apps) != 1 {
		t.Fatalf("applications = %d, want 1 (no duplicates)", len(apps))
	}

	// Cross-context read via IAM repo inside the same tenant tx still works
	// after the savepoint dance (aborted-tx regression guard).
	iamRepo := iamrepo.NewPostgresIAMRepo(pool)
	err = db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
		_, err := iamRepo.GetOrg(tctx, uuid.MustParse(orgID))
		return err
	})
	if err != nil {
		t.Fatalf("org read after screening flow: %v", err)
	}
}

var _ = jobdomain.StatusActive
var _ = scrdomain.StatusPassed

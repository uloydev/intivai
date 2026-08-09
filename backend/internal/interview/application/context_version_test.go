package application

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	ctxdomain "github.com/intivai/backend/internal/context/domain"
	ctxrepo "github.com/intivai/backend/internal/context/infrastructure/persistence"
	cvrepo "github.com/intivai/backend/internal/cv/infrastructure/persistence"
	"github.com/intivai/backend/internal/iam/infrastructure/auth"
	ivdomain "github.com/intivai/backend/internal/interview/domain"
	ivrepo "github.com/intivai/backend/internal/interview/infrastructure/persistence"
	jobrepo "github.com/intivai/backend/internal/job/infrastructure/persistence"
	scrrepo "github.com/intivai/backend/internal/screening/infrastructure/persistence"
	"github.com/intivai/backend/pkg/db"
	"github.com/intivai/backend/pkg/storage"
)

// Context version pinned at interview creation (audit): later context uploads
// bump the version for NEW interviews; the old interview keeps its pinned one.
func TestContextVersionPinnedAtCreation(t *testing.T) {
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
	appID, jobID, candID := uuid.New(), uuid.New(), uuid.New()

	err = db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
		tx, _ := db.TxFrom(tctx)
		for _, q := range []struct {
			sql  string
			args []any
		}{
			{`INSERT INTO orgs (id, name, slug) VALUES ($1,$2,$3)`, []any{orgID, "t", "cv" + orgID[:8]}},
			{`INSERT INTO jobs (id, org_id, title, description, status, created_at) VALUES ($1,$2,$3,$4,'active',NOW())`, []any{jobID, orgID, "Go Engineer", "Go"}},
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

	minio, err := storage.New(os.Getenv("TEST_MINIO_ENDPOINT"), os.Getenv("TEST_MINIO_ACCESS"), os.Getenv("TEST_MINIO_SECRET"), "intivai", false)
	if err != nil || minio == nil {
		t.Skip("TEST_MINIO_* not set")
	}
	contextRepo := ctxrepo.NewPostgresContextRepo(pool)
	svc := NewInterviewService(pool,
		ivrepo.NewPostgresInterviewRepo(pool), ivrepo.NewPostgresTokenRepo(pool), ivrepo.NewPostgresQuestionBank(pool),
		scrrepo.NewPostgresApplicationRepo(pool), cvrepo.NewPostgresCandidateRepo(pool), jobrepo.NewPostgresJobRepo(pool),
		contextRepo, minio, auth.NewJWTProvider("test-secret"), ivdomain.SystemClock())
	actor := iamActor(orgID, "admin")

	// Version 1 context → interview 1 pins version 1.
	if err := db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
		cc, err := ctxdomain.NewCompanyContext(uuid.MustParse(orgID), ctxdomain.TypeText, "hash1", "")
		if err != nil {
			return err
		}
		return contextRepo.CreateContext(tctx, cc)
	}); err != nil {
		t.Fatal(err)
	}
	iv1, err := svc.CreateInterview(ctx, actor, CreateInterviewCommand{ApplicationID: appID, QuestionCount: 2})
	if err != nil {
		t.Fatal(err)
	}
	if iv1.ContextVersion != 1 {
		t.Fatalf("interview 1 context_version = %d, want 1", iv1.ContextVersion)
	}

	// Version 2 context → interview 2 pins version 2; interview 1 unchanged.
	if err := db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
		cc, err := ctxdomain.NewCompanyContext(uuid.MustParse(orgID), ctxdomain.TypeText, "hash2", "")
		if err != nil {
			return err
		}
		cc.Version = 2
		return contextRepo.CreateContext(tctx, cc)
	}); err != nil {
		t.Fatal(err)
	}
	iv2, err := svc.CreateInterview(ctx, actor, CreateInterviewCommand{ApplicationID: appID, QuestionCount: 2})
	if err != nil {
		t.Fatal(err)
	}
	if iv2.ContextVersion != 2 {
		t.Fatalf("interview 2 context_version = %d, want 2", iv2.ContextVersion)
	}

	var v1, v2 int
	err = db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
		tx, _ := db.TxFrom(tctx)
		for _, q := range []struct {
			ivID *uuid.UUID
			dest *int
		}{{&iv1.InterviewID, &v1}, {&iv2.InterviewID, &v2}} {
			if err := tx.Raw(`SELECT context_version FROM interviews WHERE id = $1`, *q.ivID).Row().Scan(q.dest); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if v1 != 1 || v2 != 2 {
		t.Fatalf("pinned versions = (%d, %d), want (1, 2)", v1, v2)
	}
}

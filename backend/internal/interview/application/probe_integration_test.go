package application

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
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

// Shallow answer → deterministic probe follow-up on the same topic; detailed
// answer → next planned question (Research §2 weakness/strength strategy).
func TestProbeFollowUpOnShallowAnswer(t *testing.T) {
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
			{`INSERT INTO orgs (id, name, slug) VALUES ($1,$2,$3)`, []any{orgID, "t", "pr" + orgID[:8]}},
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
	svc := NewInterviewService(pool,
		ivrepo.NewPostgresInterviewRepo(pool), ivrepo.NewPostgresTokenRepo(pool), ivrepo.NewPostgresQuestionBank(pool),
		scrrepo.NewPostgresApplicationRepo(pool), cvrepo.NewPostgresCandidateRepo(pool), jobrepo.NewPostgresJobRepo(pool),
		ctxrepo.NewPostgresContextRepo(pool), minio, auth.NewJWTProvider("test-secret"), ivdomain.SystemClock())
	actor := iamActor(orgID, "admin")

	res, err := svc.CreateInterview(ctx, actor, CreateInterviewCommand{ApplicationID: appID, QuestionCount: 2})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.StartInterview(ctx, orgID, res.InterviewID); err != nil {
		t.Fatal(err)
	}

	// Q1: shallow answer → next must be a probe on the same topic.
	next, err := svc.AnswerAndAdvance(ctx, orgID, res.InterviewID, "Yes.")
	if err != nil {
		t.Fatal(err)
	}
	if next == nil || !strings.Contains(next.Content, "elaborate") {
		t.Fatalf("expected probe follow-up, got %+v", next)
	}
	firstIdx := next.Idx

	// Q2 (the probe): detailed answer → next is the original Q2 or completion.
	next, err = svc.AnswerAndAdvance(ctx, orgID, res.InterviewID,
		"I led the payment rewrite with Go, Postgres and Kafka, shipping 10k rps with three nines uptime.")
	if err != nil {
		t.Fatal(err)
	}
	if next != nil && next.Idx <= firstIdx {
		t.Fatalf("next question idx %d must follow the probe (idx %d)", next.Idx, firstIdx)
	}
}

package application

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	ctxrepo "github.com/intivai/backend/internal/context/infrastructure/persistence"
	cvrepo "github.com/intivai/backend/internal/cv/infrastructure/persistence"
	"github.com/intivai/backend/internal/iam/infrastructure/auth"
	ivdomain "github.com/intivai/backend/internal/interview/domain"
	gensvc "github.com/intivai/backend/internal/interview/domain/service"
	ivrepo "github.com/intivai/backend/internal/interview/infrastructure/persistence"
	jobrepo "github.com/intivai/backend/internal/job/infrastructure/persistence"
	scrrepo "github.com/intivai/backend/internal/screening/infrastructure/persistence"
	"github.com/intivai/backend/pkg/db"
	"github.com/intivai/backend/pkg/storage"
)

// RecentContext rebuilds the conversation history (question/answer pairs) from
// the persisted transcript, in order, windowed to the last 10 Q&A.
func TestRecentContextFromTranscript(t *testing.T) {
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
			{`INSERT INTO orgs (id, name, slug) VALUES ($1,$2,$3)`, []any{orgID, "t", "rc" + orgID[:8]}},
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
		ctxrepo.NewPostgresContextRepo(pool), minio, auth.NewJWTProvider("test-secret"), ivdomain.SystemClock(), nil)

	actor := iamActor(orgID, "admin")
	res, err := svc.CreateInterview(ctx, actor, CreateInterviewCommand{ApplicationID: appID, QuestionCount: 3})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.StartInterview(ctx, orgID, res.InterviewID); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 2; i++ {
		if _, err := svc.AnswerAndAdvance(ctx, orgID, res.InterviewID, "candidate answer"); err != nil {
			t.Fatal(err)
		}
	}

	history, err := svc.RecentContext(ctx, orgID, res.InterviewID)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 4 {
		t.Fatalf("history len = %d, want 4 (2 pairs)", len(history))
	}
	for i, m := range history {
		wantRole := gensvc.RoleAssistant
		if i%2 == 1 {
			wantRole = gensvc.RoleUser
		}
		if m.Role != wantRole {
			t.Fatalf("msg %d role = %q, want %q", i, m.Role, wantRole)
		}
	}
	if history[0].Content == "" || history[1].Content != "candidate answer" {
		t.Fatalf("pair content wrong: %+v", history)
	}
}

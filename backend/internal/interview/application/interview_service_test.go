package application

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	ctxrepo "github.com/intivai/backend/internal/context/infrastructure/persistence"
	cvrepo "github.com/intivai/backend/internal/cv/infrastructure/persistence"
	iamapp "github.com/intivai/backend/internal/iam/application"
	"github.com/intivai/backend/internal/iam/infrastructure/auth"
	ivdomain "github.com/intivai/backend/internal/interview/domain"
	ivrepo "github.com/intivai/backend/internal/interview/infrastructure/persistence"
	jobrepo "github.com/intivai/backend/internal/job/infrastructure/persistence"
	scrrepo "github.com/intivai/backend/internal/screening/infrastructure/persistence"
	"github.com/intivai/backend/pkg/db"
	"github.com/intivai/backend/pkg/storage"
)

// RED: archived jobs must not produce interviews (consistent with the
// JOB_NOT_ACTIVE rule in screening).
func TestCreateInterviewRejectsArchivedJob(t *testing.T) {
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
			{`INSERT INTO orgs (id, name, slug) VALUES ($1,$2,$3)`, []any{orgID, "t", "ar" + orgID[:8]}},
			{`INSERT INTO jobs (id, org_id, title, description, status, created_at) VALUES ($1,$2,$3,$4,'archived',NOW())`, []any{jobID, orgID, "Old Job", "desc"}},
			{`INSERT INTO candidates (id, org_id, name, email, status, created_at) VALUES ($1,$2,$3,$4,'extracted',NOW())`, []any{candID, orgID, "Jane", "j@x.io"}},
			{`INSERT INTO applications (id, org_id, candidate_id, job_id, status, cv_score, passed_screening, created_at) VALUES ($1,$2,$3,$4,'passed',80,true,NOW())`, []any{appID, orgID, candID, jobID}},
		} {
			if err := tx.Exec(q.sql, q.args...).Error; err != nil {
				return err
			}
		}
		candRepo := cvrepo.NewPostgresCandidateRepo(pool)
		c, err := candRepo.GetByID(tctx, candID)
		if err != nil {
			return err
		}
		c.CVStructured = []byte(`{"skills":["Go"],"experience_years":5,"education":"Master","certifications":[],"summary":"Go engineer"}`)
		return candRepo.Update(tctx, c)
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
	_, err = svc.CreateInterview(ctx, actor, CreateInterviewCommand{ApplicationID: appID, QuestionCount: 3})
	if err == nil || err.Error() != "job is not active" {
		t.Fatalf("archived job accepted: %v", err)
	}
}

type spyEnqueuer struct {
	lastOrgID       string
	lastInterviewID string
	lastTo          string
	lastCandName    string
	lastJobTitle    string
	lastInviteToken string
}

func (s *spyEnqueuer) EnqueueEvaluation(ctx context.Context, orgID, interviewID string) error {
	s.lastOrgID = orgID
	s.lastInterviewID = interviewID
	return nil
}

func (s *spyEnqueuer) EnqueueInterviewInvitation(ctx context.Context, to, name, jobTitle, interviewID, inviteToken string) error {
	s.lastTo = to
	s.lastCandName = name
	s.lastJobTitle = jobTitle
	s.lastInviteToken = inviteToken
	return nil
}

func TestCreateInterviewDispatchesInvitationEmail(t *testing.T) {
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
			{`INSERT INTO orgs (id, name, slug) VALUES ($1,$2,$3)`, []any{orgID, "Test Org", "to" + orgID[:8]}},
			{`INSERT INTO jobs (id, org_id, title, description, status, created_at) VALUES ($1,$2,$3,$4,'active',NOW())`, []any{jobID, orgID, "Senior Go Architect", "desc"}},
			{`INSERT INTO candidates (id, org_id, name, email, status, created_at) VALUES ($1,$2,$3,$4,'extracted',NOW())`, []any{candID, orgID, "Jane Developer", "jane@example.com"}},
			{`INSERT INTO applications (id, org_id, candidate_id, job_id, status, cv_score, passed_screening, created_at) VALUES ($1,$2,$3,$4,'passed',85,true,NOW())`, []any{appID, orgID, candID, jobID}},
		} {
			if err := tx.Exec(q.sql, q.args...).Error; err != nil {
				return err
			}
		}
		candRepo := cvrepo.NewPostgresCandidateRepo(pool)
		c, err := candRepo.GetByID(tctx, candID)
		if err != nil {
			return err
		}
		c.CVStructured = []byte(`{"skills":["Go","PostgreSQL"],"experience_years":7,"education":"BS","certifications":[],"summary":"Go architect"}`)
		return candRepo.Update(tctx, c)
	})
	if err != nil {
		t.Fatal(err)
	}

	minio, err := storage.New(os.Getenv("TEST_MINIO_ENDPOINT"), os.Getenv("TEST_MINIO_ACCESS"), os.Getenv("TEST_MINIO_SECRET"), "intivai", false)
	if err != nil || minio == nil {
		t.Skip("TEST_MINIO_* not set")
	}

	spy := &spyEnqueuer{}
	svc := NewInterviewService(pool,
		ivrepo.NewPostgresInterviewRepo(pool), ivrepo.NewPostgresTokenRepo(pool), ivrepo.NewPostgresQuestionBank(pool),
		scrrepo.NewPostgresApplicationRepo(pool), cvrepo.NewPostgresCandidateRepo(pool), jobrepo.NewPostgresJobRepo(pool),
		ctxrepo.NewPostgresContextRepo(pool), minio, auth.NewJWTProvider("test-secret"), ivdomain.SystemClock(), spy)

	actor := iamActor(orgID, "admin")
	res, err := svc.CreateInterview(ctx, actor, CreateInterviewCommand{ApplicationID: appID, QuestionCount: 3})
	if err != nil {
		t.Fatalf("CreateInterview failed: %v", err)
	}
	if res.Token == "" {
		t.Fatal("expected non-empty invitation token")
	}

	if spy.lastTo != "jane@example.com" {
		t.Fatalf("expected email to jane@example.com, got %q", spy.lastTo)
	}
	if spy.lastCandName != "Jane Developer" {
		t.Fatalf("expected candidate name Jane Developer, got %q", spy.lastCandName)
	}
	if spy.lastJobTitle != "Senior Go Architect" {
		t.Fatalf("expected job title Senior Go Architect, got %q", spy.lastJobTitle)
	}
	if spy.lastInviteToken != res.Token {
		t.Fatalf("expected invite token %q, got %q", res.Token, spy.lastInviteToken)
	}
}

func iamActor(orgID, role string) iamapp.AuthContext {
	return iamapp.AuthContext{OrgID: uuid.MustParse(orgID), Role: role}
}

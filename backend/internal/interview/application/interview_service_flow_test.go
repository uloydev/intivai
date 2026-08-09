package application

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

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
	"gorm.io/gorm"
)

type seededInterview struct {
	pool   *gorm.DB
	svc    *InterviewService
	orgID  uuid.UUID
	appID  uuid.UUID
	invite *ivdomain.InvitationToken
	ivID   uuid.UUID
}

func seedInterviewApp(t *testing.T, jobStatus string) *seededInterview {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	pool, err := db.NewPool(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	orgID := uuid.New()
	appID, jobID, candID := uuid.New(), uuid.New(), uuid.New()

	err = db.RunInTx(ctx, pool, orgID.String(), func(tctx context.Context) error {
		tx, _ := db.TxFrom(tctx)
		for _, q := range []struct {
			sql  string
			args []any
		}{
			{`INSERT INTO orgs (id, name, slug) VALUES ($1,$2,$3)`, []any{orgID, "t", "fx" + uuid.NewString()[:8]}},
			{`INSERT INTO jobs (id, org_id, title, description, status, created_at) VALUES ($1,$2,$3,$4,$5,NOW())`, []any{jobID, orgID, "Go Engineer", "Go backend work", jobStatus}},
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
	return &seededInterview{pool: pool, svc: svc, orgID: orgID, appID: appID}
}

func (s *seededInterview) create(t *testing.T) {
	t.Helper()
	created, err := s.svc.CreateInterview(context.Background(), iamActor(s.orgID.String(), "admin"), CreateInterviewCommand{ApplicationID: s.appID, QuestionCount: 3})
	if err != nil {
		t.Fatal(err)
	}
	s.ivID = created.InterviewID
	s.invite = &ivdomain.InvitationToken{Token: created.Token, InterviewID: created.InterviewID, OrgID: s.orgID}
	// The interview flow requires consent (CONSENT_REQUIRED gate).
	if err := s.svc.GiveConsent(context.Background(), s.ivID, created.Token); err != nil {
		t.Fatal(err)
	}
}

func TestIssueTicketStateMachine(t *testing.T) {
	s := seedInterviewApp(t, "active")
	s.create(t)

	// valid → ticket issued, token now used.
	tk, err := s.svc.IssueTicket(context.Background(), IssueTicketCommand{InterviewID: s.ivID, InvitationToken: s.invite.Token})
	if err != nil || tk.Ticket == "" {
		t.Fatalf("valid ticket: %v", err)
	}
	// used → reconnect still allowed (same interview).
	if _, err := s.svc.IssueTicket(context.Background(), IssueTicketCommand{InterviewID: s.ivID, InvitationToken: s.invite.Token}); err != nil {
		t.Fatalf("reconnect ticket: %v", err)
	}
	// mismatched interview → rejected.
	if _, err := s.svc.IssueTicket(context.Background(), IssueTicketCommand{InterviewID: uuid.New(), InvitationToken: s.invite.Token}); err == nil {
		t.Fatal("mismatched interview accepted")
	}
	// unknown token → rejected.
	if _, err := s.svc.IssueTicket(context.Background(), IssueTicketCommand{InterviewID: s.ivID, InvitationToken: "bogus"}); err == nil {
		t.Fatal("unknown token accepted")
	}
}

func TestIssueTicketExpiredAndRevoked(t *testing.T) {
	s := seedInterviewApp(t, "active")
	s.create(t)

	// Expire the token in place.
	err := db.RunInTx(context.Background(), s.pool, s.orgID.String(), func(tctx context.Context) error {
		tx, _ := db.TxFrom(tctx)
		return tx.Exec(`UPDATE interview_tokens SET expires_at = NOW() - interval '1 hour' WHERE token = $1`, s.invite.Token).Error
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.svc.IssueTicket(context.Background(), IssueTicketCommand{InterviewID: s.ivID, InvitationToken: s.invite.Token}); err == nil {
		t.Fatal("expired token accepted")
	}

	// Revoked → rejected even when valid.
	s2 := seedInterviewApp(t, "active")
	s2.create(t)
	err = db.RunInTx(context.Background(), s2.pool, s2.orgID.String(), func(tctx context.Context) error {
		tx, _ := db.TxFrom(tctx)
		return tx.Exec(`UPDATE interview_tokens SET revoked_at = NOW() WHERE token = $1`, s2.invite.Token).Error
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s2.svc.IssueTicket(context.Background(), IssueTicketCommand{InterviewID: s2.ivID, InvitationToken: s2.invite.Token}); err == nil {
		t.Fatal("revoked token accepted")
	}
}

func TestAnswerAndAdvanceFlow(t *testing.T) {
	s := seedInterviewApp(t, "active")
	s.create(t)

	ctx := context.Background()
	if _, err := s.svc.IssueTicket(ctx, IssueTicketCommand{InterviewID: s.ivID, InvitationToken: s.invite.Token}); err != nil {
		t.Fatal(err)
	}
	if err := s.svc.StartInterview(ctx, s.orgID.String(), s.ivID); err != nil {
		t.Fatal(err)
	}
	next, err := s.svc.AnswerAndAdvance(ctx, s.orgID.String(), s.ivID, "answer one")
	if err != nil || next == nil {
		t.Fatalf("advance: %v", err)
	}
	// Shallow answers trigger probes — the question list grows dynamically.
	if !strings.Contains(next.Content, "elaborate") {
		t.Fatalf("expected probe after shallow answer, got %q", next.Content)
	}
	if _, err := s.svc.AnswerAndAdvance(ctx, s.orgID.String(), s.ivID, "answer two"); err != nil {
		t.Fatal(err)
	}
	if _, total, status, err := s.svc.CurrentState(ctx, s.orgID.String(), s.ivID); err != nil || total != 5 || status == "" {
		t.Fatalf("state: %v total=%d (want 5: 3 planned + 2 probes)", err, total)
	}
	// Empty answer rejected.
	if _, err := s.svc.AnswerAndAdvance(ctx, s.orgID.String(), s.ivID, ""); err == nil {
		t.Fatal("empty answer accepted")
	}
}

func TestComposePromptIncludesTenantAndRails(t *testing.T) {
	s := seedInterviewApp(t, "active")
	ctx := context.Background()
	err := db.RunInTx(ctx, s.pool, s.orgID.String(), func(tctx context.Context) error {
		return ctxrepo.NewPostgresContextRepo(s.pool).SetPrompt(tctx, &ctxdomain.TenantPrompt{OrgID: s.orgID, SystemPrompt: "Interview for fintech Go roles", Version: 1})
	})
	if err != nil {
		t.Fatal(err)
	}
	prompt, err := s.svc.ComposePrompt(ctx, s.orgID)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(prompt, "Interview for fintech Go roles") {
		t.Fatal("tenant prompt missing")
	}
	if !strings.Contains(prompt, "Safety rails") || !strings.HasSuffix(prompt, "return to the interview.") {
		t.Fatal("rails not pinned last")
	}
}

func TestCreateInterviewRequiresPassedApplication(t *testing.T) {
	s := seedInterviewApp(t, "active")
	// Reject the application in place.
	err := db.RunInTx(context.Background(), s.pool, s.orgID.String(), func(tctx context.Context) error {
		tx, _ := db.TxFrom(tctx)
		return tx.Exec(`UPDATE applications SET passed_screening = false WHERE id = $1`, s.appID).Error
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.svc.CreateInterview(context.Background(), iamActor(s.orgID.String(), "admin"), CreateInterviewCommand{ApplicationID: s.appID, QuestionCount: 3}); err == nil {
		t.Fatal("unpassed application accepted")
	}
}

var _ = time.Now

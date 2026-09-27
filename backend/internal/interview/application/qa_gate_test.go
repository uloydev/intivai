package application

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	ctxrepo "github.com/intivai/backend/internal/context/infrastructure/persistence"
	cvrepo "github.com/intivai/backend/internal/cv/infrastructure/persistence"
	"github.com/intivai/backend/internal/iam/infrastructure/auth"
	iamrepo "github.com/intivai/backend/internal/iam/infrastructure/persistence"
	ivdomain "github.com/intivai/backend/internal/interview/domain"
	ivrepo "github.com/intivai/backend/internal/interview/infrastructure/persistence"
	jobdomain "github.com/intivai/backend/internal/job/domain"
	jobrepo "github.com/intivai/backend/internal/job/infrastructure/persistence"
	scrrepo "github.com/intivai/backend/internal/screening/infrastructure/persistence"
	"github.com/intivai/backend/pkg/db"
	"github.com/intivai/backend/pkg/storage"
	"github.com/rs/zerolog"
	"gorm.io/gorm"
)

// qaLimitReader adapts the IAM org repo to OrgQALimitReader (mirrors main.go).
type qaLimitReader struct {
	repo *iamrepo.PostgresIAMRepo
}

func (r qaLimitReader) CandidateQALimit(ctx context.Context, orgID uuid.UUID) (int, error) {
	org, err := r.repo.GetOrg(ctx, orgID)
	if err != nil {
		return 0, err
	}
	if org.CandidateQALimit == nil {
		return 0, errors.New("candidate_qa_limit unset")
	}
	return *org.CandidateQALimit, nil
}

// svcWithQALimit builds a second service over the same pool with the org QA
// cap reader wired (seedInterviewApp leaves it nil to exercise defaults).
func svcWithQALimit(t *testing.T, pool *gorm.DB) *InterviewService {
	t.Helper()
	minio, err := storage.New(os.Getenv("TEST_MINIO_ENDPOINT"), os.Getenv("TEST_MINIO_ACCESS"), os.Getenv("TEST_MINIO_SECRET"), "intivai", false)
	if err != nil || minio == nil {
		t.Skip("TEST_MINIO_* not set")
	}
	return NewInterviewService(pool,
		ivrepo.NewPostgresInterviewRepo(pool), ivrepo.NewPostgresTokenRepo(pool), ivrepo.NewPostgresQuestionBank(pool),
		scrrepo.NewPostgresApplicationRepo(pool), cvrepo.NewPostgresCandidateRepo(pool), jobrepo.NewPostgresJobRepo(pool),
		jobrepo.NewPostgresCandidateContextRepo(pool),
		ctxrepo.NewPostgresContextRepo(pool), minio, auth.NewJWTProvider("test-secret"), ivdomain.SystemClock(), nil,
		qaLimitReader{repo: iamrepo.NewPostgresIAMRepo(pool)}, zerolog.Nop())
}

// clampRunes bounds persisted QA text (I13) — pure table test.
func TestClampRunes(t *testing.T) {
	if got := clampRunes("hello", 10); got != "hello" {
		t.Fatalf("short string mutated: %q", got)
	}
	if got := clampRunes("hello", 5); got != "hello" {
		t.Fatalf("at-cap string mutated: %q", got)
	}
	if got := clampRunes("abcdef", 3); got != "abc" {
		t.Fatalf("clamp = %q, want abc", got)
	}
	// Rune safety: must never split a multi-byte rune.
	if got := clampRunes("aé日", 2); got != "aé" {
		t.Fatalf("multi-byte clamp = %q, want aé", got)
	}
}

// I7: expired interviews refuse candidate QA input, the refusal persists
// honest expiry state at the storage layer, and CandidateQARemaining refuses
// on the same boundary so no LLM spend is reachable.
func TestRecordCandidateQAGatesInactiveInterviews(t *testing.T) {
	s := seedInterviewApp(t)
	s.create(t)
	ctx := context.Background()
	if err := s.svc.StartInterview(ctx, s.orgID.String(), s.ivID); err != nil {
		t.Fatal(err)
	}

	if err := s.svc.RecordCandidateQA(ctx, s.orgID.String(), s.ivID, "q1", "a1"); err != nil {
		t.Fatalf("active interview refused: %v", err)
	}

	// Expire mid-interview (RLS FORCED — tenant tx).
	if err := db.RunInTx(ctx, s.pool, s.orgID.String(), func(tctx context.Context) error {
		tx, _ := db.TxFrom(tctx)
		return tx.Exec(`UPDATE interviews SET expires_at = NOW() - interval '1 hour' WHERE id = $1`, s.ivID).Error
	}); err != nil {
		t.Fatal(err)
	}

	if err := s.svc.RecordCandidateQA(ctx, s.orgID.String(), s.ivID, "q2", "a2"); !errors.Is(err, ErrInterviewNotActive) {
		t.Fatalf("expired interview accepted QA: %v", err)
	}
	if _, err := s.svc.CandidateQARemaining(ctx, s.orgID.String(), s.ivID); !errors.Is(err, ErrInterviewNotActive) {
		t.Fatalf("remaining on expired interview = %v, want ErrInterviewNotActive", err)
	}

	// Expiry persisted honestly for recruiters.
	var status string
	if err := db.RunInTx(ctx, s.pool, s.orgID.String(), func(tctx context.Context) error {
		tx, _ := db.TxFrom(tctx)
		return tx.Raw(`SELECT status FROM interviews WHERE id = $1`, s.ivID).Scan(&status).Error
	}); err != nil {
		t.Fatal(err)
	}
	if status != string(ivdomain.StatusExpired) {
		t.Fatalf("status = %s, want expired (expiry not persisted)", status)
	}
	pairs, err := s.svc.GetCandidateQA(ctx, s.orgID.String(), s.ivID)
	if err != nil || len(pairs) != 1 {
		t.Fatalf("pairs = %d (%v), want 1 (refused asks record nothing)", len(pairs), err)
	}
}

// I4/I5: with an org cap of 1 the second ask is refused by the service itself,
// nothing extra is recorded, and remaining hits zero exactly at the boundary.
func TestRecordCandidateQAEnforcesOrgCap(t *testing.T) {
	s := seedInterviewApp(t)
	s.create(t)
	ctx := context.Background()

	capped := svcWithQALimit(t, s.pool)
	if err := db.RunInTx(ctx, s.pool, s.orgID.String(), func(tctx context.Context) error {
		tx, _ := db.TxFrom(tctx)
		return tx.Exec(`UPDATE orgs SET candidate_qa_limit = 1 WHERE id = $1`, s.orgID).Error
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.svc.StartInterview(ctx, s.orgID.String(), s.ivID); err != nil {
		t.Fatal(err)
	}

	if err := capped.RecordCandidateQA(ctx, s.orgID.String(), s.ivID, "first", "a"); err != nil {
		t.Fatalf("first ask under cap refused: %v", err)
	}
	if err := capped.RecordCandidateQA(ctx, s.orgID.String(), s.ivID, "second", "a"); !errors.Is(err, ErrQALimitExceeded) {
		t.Fatalf("second ask over cap = %v, want ErrQALimitExceeded", err)
	}
	rem, err := capped.CandidateQARemaining(ctx, s.orgID.String(), s.ivID)
	if err != nil || rem != 0 {
		t.Fatalf("remaining = %d (%v), want 0 at the org cap boundary", rem, err)
	}
	pairs, err := capped.GetCandidateQA(ctx, s.orgID.String(), s.ivID)
	if err != nil || len(pairs) != 1 || pairs[0].Question != "first" {
		t.Fatalf("pairs = %+v (%v), want exactly [first]", pairs, err)
	}
}

// ComposeConnectContexts must carry the pinned per-job candidate context into
// the QA grounding (I12 single-load contract, job-context branch).
func TestComposeConnectContextsIncludesJobCandidateContext(t *testing.T) {
	s := seedInterviewApp(t)
	s.create(t)
	ctx := context.Background()

	var jobID uuid.UUID
	if err := db.RunInTx(ctx, s.pool, s.orgID.String(), func(tctx context.Context) error {
		tx, _ := db.TxFrom(tctx)
		return tx.Raw(`SELECT job_id FROM applications WHERE id = $1`, s.appID).Row().Scan(&jobID)
	}); err != nil {
		t.Fatal(err)
	}
	cc, err := jobdomain.NewCandidateContext(s.orgID, jobID, "Benefits include unlimited PTO and a learning budget.")
	if err != nil {
		t.Fatal(err)
	}
	if err := db.RunInTx(ctx, s.pool, s.orgID.String(), func(tctx context.Context) error {
		return jobrepo.NewPostgresCandidateContextRepo(s.pool).Upsert(tctx, cc)
	}); err != nil {
		t.Fatal(err)
	}

	out, err := s.svc.ComposeConnectContexts(ctx, s.orgID, s.ivID)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.QAContext, "unlimited PTO") {
		t.Fatalf("job candidate context missing from QA grounding: %q", out.QAContext)
	}
}

package application

import (
	"bytes"
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
	"github.com/rs/zerolog"
	"gorm.io/gorm"
)

// failingBank injects QuestionBank persistence failures (D25 probe path).
type failingBank struct{ ivdomain.QuestionBank }

func (failingBank) Create(context.Context, uuid.UUID, ivdomain.Question) error {
	return errBankDown
}

var errBankDown = errQuestionBankDown{}

type errQuestionBankDown struct{}

func (errQuestionBankDown) Error() string { return "question bank unavailable" }

// seedTransitionOrg creates org + active job + passed application + structured
// candidate and returns the pool plus the ids needed to drive dialogue turns.
func seedTransitionOrg(t *testing.T) (*gorm.DB, string, uuid.UUID) {
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
	orgSlug := "tr" + uuid.NewString()[:8]
	appID, jobID, candID := uuid.New(), uuid.New(), uuid.New()

	err = db.RunInTx(ctx, pool, orgID.String(), func(tctx context.Context) error {
		tx, _ := db.TxFrom(tctx)
		for _, q := range []struct {
			sql  string
			args []any
		}{
			{`INSERT INTO orgs (id, name, slug) VALUES ($1,$2,$3)`, []any{orgID, "t", orgSlug}},
			{`INSERT INTO jobs (id, org_id, title, description, status, created_at) VALUES ($1,$2,$3,$4,'active',NOW())`, []any{jobID, orgID, "Go Engineer", "Go backend work"}},
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
	return pool, orgID.String(), appID
}

// newTransitionService builds a service whose logs land in logBuf.
func newTransitionService(pool *gorm.DB, bank ivdomain.QuestionBank, logBuf *bytes.Buffer) *InterviewService {
	minio, err := storage.New(os.Getenv("TEST_MINIO_ENDPOINT"), os.Getenv("TEST_MINIO_ACCESS"), os.Getenv("TEST_MINIO_SECRET"), "intivai", false)
	if err != nil || minio == nil {
		logBuf.Reset()
		return nil
	}
	logger := zerolog.New(logBuf)
	if logBuf == nil {
		logger = zerolog.Nop()
	}
	return NewInterviewService(pool,
		ivrepo.NewPostgresInterviewRepo(pool), ivrepo.NewPostgresTokenRepo(pool), bank,
		scrrepo.NewPostgresApplicationRepo(pool), cvrepo.NewPostgresCandidateRepo(pool), jobrepo.NewPostgresJobRepo(pool),
		jobrepo.NewPostgresCandidateContextRepo(pool),
		ctxrepo.NewPostgresContextRepo(pool), minio, auth.NewJWTProvider("test-secret"), ivdomain.SystemClock(), nil, nil, logger)
}

// createStartedInterview — interview with consent recorded and started, so
// ProcessTopicDialogue can drive the in_progress state machine directly.
func createStartedInterview(t *testing.T, svc *InterviewService, orgIDStr string, appID uuid.UUID, questionCount int) *CreateInterviewResult {
	t.Helper()
	created, err := svc.CreateInterview(context.Background(), iamActor(orgIDStr), CreateInterviewCommand{ApplicationID: appID, QuestionCount: questionCount})
	if err != nil {
		t.Fatalf("create interview: %v", err)
	}
	if err := svc.GiveConsent(context.Background(), created.InterviewID, created.Token); err != nil {
		t.Fatalf("consent: %v", err)
	}
	if err := svc.StartInterview(context.Background(), orgIDStr, created.InterviewID); err != nil {
		t.Fatalf("start interview: %v", err)
	}
	return created
}

// RED (D25): a probe persist failure must be logged with context — today the
// error is silently dropped and the interview continues with no trace.
func TestProcessTopicDialogueLogsProbePersistFailure(t *testing.T) {
	pool, orgIDStr, appID := seedTransitionOrg(t)

	var logBuf bytes.Buffer
	svc := newTransitionService(pool, ivrepo.NewPostgresQuestionBank(pool), &logBuf)
	if svc == nil {
		t.Skip("TEST_MINIO_* not set")
	}

	created := createStartedInterview(t, svc, orgIDStr, appID, 3)
	// Swap in the failing bank AFTER creation — CreateInterview itself
	// persists the generated questions through it.
	svc.bank = failingBank{ivrepo.NewPostgresQuestionBank(pool)}

	// Short answer + advance completes the topic → ShouldProbe fires, the
	// probe insert succeeds in-memory but its persistence must fail.
	res, err := svc.ProcessTopicDialogue(context.Background(), orgIDStr, created.InterviewID, "brief", "advance", nil)
	if err != nil {
		t.Fatalf("process topic dialogue: %v", err)
	}
	if res.NextQuestion == nil || res.NextQuestion.IsProbe {
		t.Fatalf("expected original next question without probe after failed persist, got %+v", res.NextQuestion)
	}

	logs := logBuf.String()
	if !strings.Contains(logs, created.InterviewID.String()) ||
		!strings.Contains(logs, "probe_persist") ||
		!strings.Contains(logs, errBankDown.Error()) {
		t.Fatalf("probe persist failure swallowed — expected contextual error log with interview id and stage, got %q", logs)
	}
}

var errCompleteDown = errCompleteRefused{}

type errCompleteRefused struct{}

func (errCompleteRefused) Error() string { return "transition refused" }

// RED->GREEN (D25): a failing Complete transition during advance must be
// surfaced on the result (the WS layer emits it through the standard
// error-frame path) AND logged — while the next question still dispatches
// (LLM-failure advance semantics preserved).
func TestProcessTopicDialogueSurfacesCompleteFailure(t *testing.T) {
	pool, orgIDStr, appID := seedTransitionOrg(t)

	var logBuf bytes.Buffer
	svc := newTransitionService(pool, ivrepo.NewPostgresQuestionBank(pool), &logBuf)
	if svc == nil {
		t.Skip("TEST_MINIO_* not set")
	}

	// Single question: this one advance exhausts the interview, so the
	// Complete transition actually runs (and fails). The answer is detailed
	// enough to skip the weakness-probe path — the probe would keep a next
	// question alive and bypass Complete entirely.
	created := createStartedInterview(t, svc, orgIDStr, appID, 1)

	svc.completeFn = func(*ivdomain.Interview) error { return errCompleteDown }

	res, err := svc.ProcessTopicDialogue(context.Background(), orgIDStr, created.InterviewID,
		"I architected our payment platform on Go microservices with Postgres and Kafka, owning schema design and rollouts", "advance", nil)
	if err != nil {
		t.Fatalf("process topic dialogue failed outright: %v", err)
	}
	// The dialogue result must survive: topic finalized, no error abort —
	// the caller (WS layer) keeps moving and dispatches what follows
	// (evaluation when the interview is exhausted).
	if !res.IsTopicComplete {
		t.Fatalf("expected completed topic, got %+v", res)
	}
	if res.TransitionErr == nil {
		t.Fatal("Complete failure silently discarded — expected TransitionErr for the WS error frame path")
	}

	logs := logBuf.String()
	if !strings.Contains(logs, created.InterviewID.String()) ||
		!strings.Contains(logs, errCompleteDown.Error()) {
		t.Fatalf("complete failure not logged with context, got %q", logs)
	}
}

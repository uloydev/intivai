package application

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	cvdomain "github.com/intivai/backend/internal/cv/domain"
	cvrepo "github.com/intivai/backend/internal/cv/infrastructure/persistence"
	jobrepo "github.com/intivai/backend/internal/job/infrastructure/persistence"
	scrdomain "github.com/intivai/backend/internal/screening/domain"
	scrrepo "github.com/intivai/backend/internal/screening/infrastructure/persistence"
	shareddomain "github.com/intivai/backend/internal/shared/domain"
	"github.com/intivai/backend/pkg/db"
	"github.com/rs/zerolog"
)

// Integration test — live Postgres + Redis required.
// Guards the score worker end-to-end: structured candidate → application →
// weighted score persisted (status passed/rejected + breakdown).
func TestScoreWorkerPipeline(t *testing.T) {
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

	seed := func() {
		t.Helper()
		err = db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
			tx, ok := db.TxFrom(tctx)
			if !ok {
				return db.ErrNoTx
			}
			if err := tx.Exec(`INSERT INTO orgs (id, name, slug) VALUES ($1,$2,$3)`, orgID, "t", "s"+orgID[:8]).Error; err != nil {
				return err
			}
			if err := tx.Exec(`INSERT INTO jobs (id, org_id, title, description, status, created_at) VALUES ($1,$2,$3,$4,'active',NOW())`, jobID, orgID, "Go Engineer", "Go backend work").Error; err != nil {
				return err
			}
			candRepo := cvrepo.NewPostgresCandidateRepo(pool)
			c := &cvdomain.Candidate{
				Entity: shareddomain.Entity{ID: candID, CreatedAt: time.Now().UTC()},
				OrgID:  uuid.MustParse(orgID), Name: "Jane", Status: "parsed",
			}
			if err := candRepo.Create(tctx, c); err != nil {
				return err
			}
			c.CVStructured = []byte(`{"skills":["Go","PostgreSQL"],"experience_years":5,"education":"Master","certifications":["AWS"],"summary":"Go engineer"}`)
			c.Status = "extracted"
			return candRepo.Update(tctx, c)
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	seed()

	appRepo := scrrepo.NewPostgresApplicationRepo(pool)
	var app *scrdomain.Application
	err = db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
		app = scrdomain.NewApplication(uuid.MustParse(orgID), candID, jobID)
		return appRepo.Create(tctx, app)
	})
	if err != nil {
		t.Fatal(err)
	}

	worker := NewScoreWorker(pool, appRepo, cvrepo.NewPostgresCandidateRepo(pool), jobrepo.NewPostgresJobRepo(pool), stubOrgSettings{orgID: orgID}, nil, testLogger())
	payload, _ := json.Marshal(ScoreCVPayload{OrgID: orgID, ApplicationID: app.ID.String()})
	if err := worker.handle(ctx, asynq.NewTask(TaskScoreCV, payload)); err != nil {
		t.Fatalf("score worker: %v", err)
	}

	var scored *scrdomain.Application
	err = db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
		scored, err = appRepo.GetByID(tctx, app.ID)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if scored.CVScore == nil || scored.PassedScreening == nil {
		t.Fatalf("score not persisted: %+v", scored)
	}
	if scored.Status != scrdomain.StatusPassed && scored.Status != scrdomain.StatusRejected {
		t.Fatalf("status = %q", scored.Status)
	}
	if len(scored.ScoreBreakdown) == 0 {
		t.Fatal("breakdown missing")
	}
	t.Logf("score=%v passed=%v breakdown=%s", *scored.CVScore, *scored.PassedScreening, scored.ScoreBreakdown)

	// Idempotent re-run: same result, no error.
	before := *scored.CVScore
	if err := worker.handle(ctx, asynq.NewTask(TaskScoreCV, payload)); err != nil {
		t.Fatalf("re-score: %v", err)
	}
	err = db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
		scored, err = appRepo.GetByID(tctx, app.ID)
		return err
	})
	if err != nil || *scored.CVScore != before {
		t.Fatalf("re-score changed result: %v vs %v", before, *scored.CVScore)
	}
}

type stubOrgSettings struct {
	orgID string
}

func (s stubOrgSettings) ReadOrgSettings(ctx context.Context, orgID uuid.UUID) (map[string]float64, float64, error) {
	return map[string]float64{}, 50, nil
}

func testLogger() zerolog.Logger { return zerolog.Nop() }

package persistence

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	ivdomain "github.com/intivai/backend/internal/interview/domain"
	"github.com/intivai/backend/pkg/db"
	"github.com/stretchr/testify/assert"
)

func TestProctoringRepoRoundTrip(t *testing.T) {
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
	seedOrg(t, pool, orgID, "proc"+orgID[:8])

	jobID := uuid.New()
	candID := uuid.New()
	appID := uuid.New()
	err = db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
		tx, _ := db.TxFrom(tctx)
		if err := tx.Exec(`INSERT INTO jobs (id, org_id, title, description, status, created_at) VALUES ($1,$2,$3,$4,'active',NOW())`, jobID, orgID, "Job", "desc").Error; err != nil {
			return err
		}
		if err := tx.Exec(`INSERT INTO candidates (id, org_id, name, status, created_at) VALUES ($1,$2,$3,'extracted',NOW())`, candID, orgID, "Candidate").Error; err != nil {
			return err
		}
		return tx.Exec(`INSERT INTO applications (id, org_id, candidate_id, job_id, status, created_at) VALUES ($1,$2,$3,$4,'passed',NOW())`, appID, orgID, candID, jobID).Error
	})
	if err != nil {
		t.Fatal(err)
	}

	repo := NewPostgresInterviewRepo(pool)
	iv, err := ivdomain.NewInterview(uuid.MustParse(orgID), appID, []ivdomain.Question{
		{Idx: 1, Content: "Q1", Category: "technical", Skill: "Go"},
	}, time.Now().Add(24*time.Hour), ivdomain.SystemClock())
	if err != nil {
		t.Fatal(err)
	}

	// 1. Create interview
	err = db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
		return repo.Create(tctx, iv)
	})
	assert.NoError(t, err)

	// 2. Record telemetry events
	now := time.Now().UTC().Truncate(time.Millisecond)
	err = db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
		ev1 := ivdomain.ProctoringEvent{
			Type:        ivdomain.EventTypeTabSwitch,
			Timestamp:   now,
			QuestionIdx: 1,
		}
		if err := repo.RecordProctoringEvent(tctx, iv.ID, ev1); err != nil {
			return err
		}
		ev2 := ivdomain.ProctoringEvent{
			Type:        ivdomain.EventTypePaste,
			Timestamp:   now.Add(5 * time.Second),
			QuestionIdx: 1,
			Details:     &ivdomain.TelemetryDetails{PastedTextLength: 240},
		}
		return repo.RecordProctoringEvent(tctx, iv.ID, ev2)
	})
	assert.NoError(t, err)

	// 3. Hydrate and verify proctoring telemetry
	var loaded *ivdomain.Interview
	err = db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
		var err error
		loaded, err = repo.GetByID(tctx, iv.ID)
		return err
	})
	assert.NoError(t, err)
	assert.NotNil(t, loaded)
	assert.Len(t, loaded.ProctoringEvents, 2)
	assert.Equal(t, ivdomain.EventTypeTabSwitch, loaded.ProctoringEvents[0].Type)
	assert.Equal(t, ivdomain.EventTypePaste, loaded.ProctoringEvents[1].Type)
	assert.Equal(t, 1, loaded.ProctoringSummary.TabSwitchCount)
	assert.Equal(t, 1, loaded.ProctoringSummary.SuspiciousPasteCount)
	// 100 - 5 (tab switch) - 15 (suspicious paste) = 80
	assert.Equal(t, 80, loaded.ProctoringSummary.IntegrityScore)
	assert.Equal(t, ivdomain.RiskMedium, loaded.ProctoringSummary.RiskLevel)
	assert.NotEmpty(t, loaded.ProctoringSummary.Flags)
}

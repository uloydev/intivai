package application_test

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/intivai/backend/internal/iam/application"
	"github.com/intivai/backend/internal/iam/domain"
	scrapp "github.com/intivai/backend/internal/screening/application"
	scrdomain "github.com/intivai/backend/internal/screening/domain"
	"github.com/intivai/backend/internal/screening/infrastructure/persistence"
	"github.com/intivai/backend/pkg/db"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestScreeningService_UpdateDecisionTransitions(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("skipping integration test; TEST_DATABASE_URL not set")
	}
	pool, err := gorm.Open(postgres.Open(dbURL), &gorm.Config{})
	require.NoError(t, err)

	repo := persistence.NewPostgresApplicationRepo(pool)
	svc := scrapp.NewScreeningService(pool, repo, nil, nil, nil)

	orgID := uuid.New()
	candID := uuid.New()
	jobID := uuid.New()
	appID := uuid.New()

	ctx := context.Background()

	// 1. Setup seed data
	err = db.RunInTx(ctx, pool, orgID.String(), func(tx context.Context) error {
		gormTx, ok := db.TxFrom(tx)
		require.True(t, ok)
		_ = gormTx.Exec(`INSERT INTO orgs (id, name, slug) VALUES (?, ?, ?) ON CONFLICT DO NOTHING`, orgID, "Test Org", "test-"+uuid.NewString()[:6])
		_ = gormTx.Exec(`INSERT INTO jobs (id, org_id, title, description, location, employment_type) VALUES (?, ?, ?, ?, ?, ?)`,
			jobID, orgID, "Test Job", "Test Desc", "Remote", "Full-time")
		_ = gormTx.Exec(`INSERT INTO candidates (id, org_id, name, email) VALUES (?, ?, ?, ?)`,
			candID, orgID, "Jane Candidate", "jane@test.io")
		_ = gormTx.Exec(`INSERT INTO applications (id, org_id, candidate_id, job_id, status, stage) VALUES (?, ?, ?, ?, ?, ?)`,
			appID, orgID, candID, jobID, "screening", "applied")
		return nil
	})
	require.NoError(t, err)

	recruiter := application.AuthContext{
		UserID: uuid.New(),
		OrgID:  orgID,
		Role:   string(domain.RoleRecruiter),
	}

	admin := application.AuthContext{
		UserID: uuid.New(),
		OrgID:  orgID,
		Role:   string(domain.RoleAdmin),
	}

	// 2. Normal forward transition (Recruiter)
	stageInterview := string(scrdomain.StageInterviewInvited)
	res, err := svc.UpdateDecision(ctx, recruiter, appID, &stageInterview, nil)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, stageInterview, *res.Stage)

	// 3. Backward transition rejected (Recruiter)
	stageApplied := string(scrdomain.StageApplied)
	_, err = svc.UpdateDecision(ctx, recruiter, appID, &stageApplied, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "transition from \"interview_invited\" to \"applied\" is not allowed")

	// 4. Backward transition accepted (Admin)
	res, err = svc.UpdateDecision(ctx, admin, appID, &stageApplied, nil)
	require.NoError(t, err)
	require.Equal(t, stageApplied, *res.Stage)

	// 5. Terminal state allowed from anywhere
	stageRejected := string(scrdomain.StageRejected)
	res, err = svc.UpdateDecision(ctx, recruiter, appID, &stageRejected, nil)
	require.NoError(t, err)
	require.Equal(t, stageRejected, *res.Stage)

	// 6. Terminal reversal rejected (Recruiter)
	_, err = svc.UpdateDecision(ctx, recruiter, appID, &stageApplied, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "transition from \"rejected\" to \"applied\" is not allowed")

	// 7. Terminal reversal accepted (Admin)
	res, err = svc.UpdateDecision(ctx, admin, appID, &stageApplied, nil)
	require.NoError(t, err)
	require.Equal(t, stageApplied, *res.Stage)
}

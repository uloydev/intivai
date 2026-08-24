package persistence_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	scrdomain "github.com/intivai/backend/internal/screening/domain"
	scrrepo "github.com/intivai/backend/internal/screening/infrastructure/persistence"
	"github.com/intivai/backend/pkg/db"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func newPortalTestPool(t *testing.T) *gorm.DB {
	t.Helper()
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("skipping integration test; TEST_DATABASE_URL not set")
	}
	pool, err := gorm.Open(postgres.Open(dbURL), &gorm.Config{})
	require.NoError(t, err)
	return pool
}

// TestApplyWithDedupe_ReapplyNewJobCreatesApplication — re-applying with the
// same email to a DIFFERENT job must link the existing candidate to the new
// job idempotently (doc contract: "then insert the application
// idempotently"). Without this the per-email abuse cap cannot observe
// multi-job apply spam.
func TestApplyWithDedupe_ReapplyNewJobCreatesApplication(t *testing.T) {
	pool := newPortalTestPool(t)
	ctx := context.Background()

	orgID := uuid.New()
	jobA := uuid.New()
	jobB := uuid.New()
	email := "dedupe-cap-" + uuid.NewString()[:8] + "@candidate.io"

	for _, jid := range []uuid.UUID{jobA, jobB} {
		err := db.RunInTx(ctx, pool, orgID.String(), func(tctx context.Context) error {
			tx, ok := db.TxFrom(tctx)
			if !ok {
				t.Fatal("no tenant tx")
			}
			if err := tx.Exec(`INSERT INTO orgs (id, name, slug) VALUES (?, ?, ?) ON CONFLICT (id) DO NOTHING`,
				orgID, "Dedupe Co", "dedupe-"+orgID.String()[:8]).Error; err != nil {
				return err
			}
			return tx.Exec(`INSERT INTO jobs (id, org_id, title, description, location, employment_type) VALUES (?, ?, ?, ?, ?, ?)`,
				jid, orgID, "Dedupe Engineer", "d", "Remote", "Full-time").Error
		})
		require.NoError(t, err)
	}

	repo := scrrepo.NewPostgresApplicationRepo(pool)
	appRepo := scrdomain.ApplicationRepository(repo)

	var candA uuid.UUID
	err := db.RunInTx(ctx, pool, orgID.String(), func(tctx context.Context) error {
		id, isNew, err := appRepo.ApplyWithDedupe(tctx, orgID, jobA, "Dedupe Candidate", email)
		candA = id
		if err != nil {
			return err
		}
		require.True(t, isNew, "first apply creates the candidate")
		return nil
	})
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, candA)

	var candB uuid.UUID
	var isNewB bool
	err = db.RunInTx(ctx, pool, orgID.String(), func(tctx context.Context) error {
		id, isNew, err := appRepo.ApplyWithDedupe(tctx, orgID, jobB, "Dedupe Candidate", email)
		candB, isNewB = id, isNew
		return err
	})
	require.NoError(t, err)
	require.False(t, isNewB, "second apply reuses the candidate")
	require.Equal(t, candA, candB)

	// Both applications exist and resolve.
	var a1, a2 *scrdomain.Application
	err = db.RunInTx(ctx, pool, orgID.String(), func(tctx context.Context) error {
		var e1, e2 error
		a1, e1 = appRepo.GetByCandidateJob(tctx, orgID, candA, jobA)
		if e1 != nil {
			return e1
		}
		a2, e2 = appRepo.GetByCandidateJob(tctx, orgID, candA, jobB)
		return e2
	})
	require.NoError(t, err)
	require.NotNil(t, a1)
	require.NotNil(t, a2, "re-apply to a new job must create its application row")
}

// TestCandidateApplicationsLookup_StripsInvitationToken — C9 hardening: the
// candidate-portal lookup function must resolve application rows WITHOUT
// exposing interview invitation_token. The token is a live interview
// credential; portal JWT holders must never be able to read it.
func TestCandidateApplicationsLookup_StripsInvitationToken(t *testing.T) {
	pool := newPortalTestPool(t)
	ctx := context.Background()

	orgID := uuid.New()
	jobID := uuid.New()
	candID := uuid.New()
	appID := uuid.New()
	email := "lookup-strip-" + uuid.NewString()[:8] + "@candidate.io"
	jobTitle := "Strip Probe Engineer"

	err := db.RunInTx(ctx, pool, orgID.String(), func(tctx context.Context) error {
		tx, ok := db.TxFrom(tctx)
		if !ok {
			t.Fatal("no tenant tx")
		}
		if err := tx.Exec(`INSERT INTO orgs (id, name, slug) VALUES (?, ?, ?)`,
			orgID, "Strip Co", "strip-"+uuid.NewString()[:6]).Error; err != nil {
			return err
		}
		if err := tx.Exec(`INSERT INTO jobs (id, org_id, title, description, location, employment_type) VALUES (?, ?, ?, ?, ?, ?)`,
			jobID, orgID, jobTitle, "lookup strip", "Remote", "Full-time").Error; err != nil {
			return err
		}
		if err := tx.Exec(`INSERT INTO candidates (id, org_id, name, email) VALUES (?, ?, ?, ?)`,
			candID, orgID, "Strip Candidate", email).Error; err != nil {
			return err
		}
		return tx.Exec(`INSERT INTO applications (id, org_id, candidate_id, job_id, status) VALUES (?, ?, ?, ?, 'screening')`,
			appID, orgID, candID, jobID).Error
	})
	require.NoError(t, err)

	repo := scrrepo.NewPostgresCandidatePortalRepo(pool)

	// Portal flows still resolve the application (all other columns intact).
	views, err := repo.ListApplications(ctx, email)
	require.NoError(t, err)
	require.Len(t, views, 1)
	require.Equal(t, appID, views[0].ApplicationID)
	require.Equal(t, jobTitle, views[0].JobTitle)
	require.Equal(t, "Strip Co", views[0].OrgName)

	// The credential column must no longer exist on the function output.
	var probe string
	err = pool.WithContext(ctx).Raw(
		`SELECT invitation_token FROM candidate_applications_lookup(?) LIMIT 1`, email,
	).Row().Scan(&probe)
	require.Error(t, err, "invitation_token must not be selectable from candidate_applications_lookup")
	require.True(t, strings.Contains(err.Error(), "does not exist"), "expected missing-column error, got: %v", err)
}

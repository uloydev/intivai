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

// insertDemoOTP — inserts a reusable "demo-" magic-token row for a demo user.
func insertDemoOTP(t *testing.T, pool *gorm.DB, email, token string) {
	t.Helper()
	err := pool.Exec(
		`INSERT INTO candidate_otps (id, email, code_hash, token, attempts, expires_at, created_at)
		 VALUES (?, ?, ?, ?, 0, NOW() + INTERVAL '24 hours', NOW())`,
		uuid.New(), email, uuid.NewString(), token,
	).Error
	require.NoError(t, err)
}

// TestDemoMagicTokenRejectedWithoutDemoEnv — D22 regression: with
// INTIVAI_DEMO_TOKENS unset (production config), a "demo-" token must behave
// like a normal single-use token: FindValidByToken rejects a consumed demo
// token and Consume marks it used once (second consume returns false).
func TestDemoMagicTokenRejectedWithoutDemoEnv(t *testing.T) {
	t.Setenv("INTIVAI_DEMO_TOKENS", "")
	pool := newPortalTestPool(t)
	ctx := context.Background()

	email := "demo-prod-" + uuid.NewString()[:8] + "@demo.io"
	token := "demo-prod-token-" + uuid.NewString()[:8]
	insertDemoOTP(t, pool, email, token)

	repo := scrrepo.NewPostgresCandidatePortalRepo(pool)

	// Not-yet-used demo token: must resolve.
	otp, err := repo.FindValidByToken(ctx, token)
	require.NoError(t, err)
	require.NotNil(t, otp, "unused demo token must resolve without demo env (single-use semantics)")

	// First consume: marks used and returns true.
	consumed, err := repo.Consume(ctx, otp.ID)
	require.NoError(t, err)
	require.True(t, consumed, "first consume must succeed even without demo env")

	// Second consume: the row is used_at-marked, so it must NOT pass again
	// (demo tokens must not be replayable in prod config).
	consumed, err = repo.Consume(ctx, otp.ID)
	require.NoError(t, err)
	require.False(t, consumed, "replay of a demo token must be rejected without demo env")

	// A consumed demo token must no longer resolve via FindValidByToken.
	otp, err = repo.FindValidByToken(ctx, token)
	require.NoError(t, err)
	require.Nil(t, otp, "consumed demo token must not resolve without demo env")
}

// TestDemoMagicTokenReusedWhenDemoEnvOn — D22: with INTIVAI_DEMO_TOKENS=1 the
// curated demo tokens stay reusable (skip used_at in lookup, no-op consume),
// so the seeded demo link keeps working during local demos.
func TestDemoMagicTokenReusedWhenDemoEnvOn(t *testing.T) {
	t.Setenv("INTIVAI_DEMO_TOKENS", "1")
	pool := newPortalTestPool(t)
	ctx := context.Background()

	email := "demo-on-" + uuid.NewString()[:8] + "@demo.io"
	token := "demo-on-token-" + uuid.NewString()[:8]
	insertDemoOTP(t, pool, email, token)

	repo := scrrepo.NewPostgresCandidatePortalRepo(pool)

	// Resolves; consume returns true; still resolves afterwards (reusable).
	otp, err := repo.FindValidByToken(ctx, token)
	require.NoError(t, err)
	require.NotNil(t, otp)
	consumed, err := repo.Consume(ctx, otp.ID)
	require.NoError(t, err)
	require.True(t, consumed)
	otp2, err := repo.FindValidByToken(ctx, token)
	require.NoError(t, err)
	require.NotNil(t, otp2, "demo token must remain reusable with demo env on")

	// A non-demo token still follows single-use semantics with demo env on.
	normalToken := "normal-" + uuid.NewString()[:8]
	normalEmail := "demo-on-" + uuid.NewString()[:8] + "@demo.io"
	insertDemoOTP(t, pool, normalEmail, normalToken)
	normalOTP, err := repo.FindValidByToken(ctx, normalToken)
	require.NoError(t, err)
	require.NotNil(t, normalOTP)
	consumed, err = repo.Consume(ctx, normalOTP.ID)
	require.NoError(t, err)
	require.True(t, consumed)
	replay, err := repo.FindValidByToken(ctx, normalToken)
	require.NoError(t, err)
	require.Nil(t, replay, "non-demo tokens must stay single-use even with demo env on")
}

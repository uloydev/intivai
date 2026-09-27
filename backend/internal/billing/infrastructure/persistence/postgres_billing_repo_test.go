package persistence_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	billingdomain "github.com/intivai/backend/internal/billing/domain"
	billingrepo "github.com/intivai/backend/internal/billing/infrastructure/persistence"
	"github.com/intivai/backend/pkg/db"
	"github.com/stretchr/testify/require"
)

func TestPostgresBillingRepo(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}

	pool, err := db.NewPool(context.Background(), url)
	require.NoError(t, err)

	repo := billingrepo.NewPostgresBillingRepo(pool)
	orgID := uuid.New()

	ctx := context.Background()
	err = db.RunInTx(ctx, pool, orgID.String(), func(tctx context.Context) error {
		tx, ok := db.TxFrom(tctx)
		require.True(t, ok)
		_ = tx.Exec(`INSERT INTO orgs (id, name, slug) VALUES (?, ?, ?) ON CONFLICT DO NOTHING`, orgID, "Billing Org", "billing-"+uuid.NewString()[:6])

		// 1. Get default billing
		b, err := repo.GetOrgBilling(tctx, orgID)
		require.NoError(t, err)
		require.Equal(t, billingdomain.PlanFree, b.Plan)
		require.Equal(t, "active", b.PlanStatus)
		require.Equal(t, 0, b.InterviewCredits)

		// 2. Update billing
		now := time.Now().UTC()
		end := now.Add(30 * 24 * time.Hour)
		b.Plan = billingdomain.PlanPro
		b.PlanStatus = "active"
		b.StripeCustomerID = "cus_test_123"
		b.StripeSubscriptionID = "sub_test_123"
		b.CurrentPeriodStart = &now
		b.CurrentPeriodEnd = &end
		b.InterviewCredits = 5

		err = repo.UpdateSubscription(tctx, b)
		require.NoError(t, err)

		bUpdated, err := repo.GetOrgBilling(tctx, orgID)
		require.NoError(t, err)
		require.Equal(t, billingdomain.PlanPro, bUpdated.Plan)
		require.Equal(t, 5, bUpdated.InterviewCredits)

		// 3. Deduct credit
		deducted, err := repo.DeductCredit(tctx, orgID)
		require.NoError(t, err)
		require.True(t, deducted)

		bAfter, err := repo.GetOrgBilling(tctx, orgID)
		require.NoError(t, err)
		require.Equal(t, 4, bAfter.InterviewCredits)

		// 4. Monthly usage query
		usage, err := repo.GetMonthlyUsage(tctx, orgID, &now)
		require.NoError(t, err)
		require.Equal(t, 0, usage)

		return nil
	})
	require.NoError(t, err)
}

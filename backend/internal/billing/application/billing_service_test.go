package application_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	billingapp "github.com/intivai/backend/internal/billing/application"
	billingdomain "github.com/intivai/backend/internal/billing/domain"
	billinggateway "github.com/intivai/backend/internal/billing/infrastructure/gateway"
	sharederr "github.com/intivai/backend/internal/shared/errors"
	"github.com/stretchr/testify/require"
)

type mockBillingRepo struct {
	billing *billingdomain.OrgBilling
	usage   int
}

func (m *mockBillingRepo) GetOrgBilling(ctx context.Context, orgID uuid.UUID) (*billingdomain.OrgBilling, error) {
	return m.billing, nil
}

func (m *mockBillingRepo) UpdateSubscription(ctx context.Context, billing *billingdomain.OrgBilling) error {
	m.billing = billing
	return nil
}

func (m *mockBillingRepo) GetMonthlyUsage(ctx context.Context, orgID uuid.UUID, since *time.Time) (int, error) {
	return m.usage, nil
}

func (m *mockBillingRepo) DeductCredit(ctx context.Context, orgID uuid.UUID) (bool, error) {
	if m.billing.InterviewCredits > 0 {
		m.billing.InterviewCredits--
		return true, nil
	}
	return false, nil
}

func TestBillingService_CheckQuota(t *testing.T) {
	orgID := uuid.New()
	gw := billinggateway.NewNoopPaymentGateway()

	t.Run("within free plan limit", func(t *testing.T) {
		repo := &mockBillingRepo{
			billing: &billingdomain.OrgBilling{
				OrgID:      orgID,
				Plan:       billingdomain.PlanFree,
				PlanStatus: "active",
			},
			usage: 5,
		}
		svc := billingapp.NewBillingService(nil, repo, gw)
		allowed, err := svc.CheckAndConsumeQuota(context.Background(), orgID)
		require.NoError(t, err)
		require.True(t, allowed)

		summary, err := svc.GetSummary(context.Background(), orgID)
		require.NoError(t, err)
		require.Equal(t, 10, summary.MonthlyLimit)
		require.Equal(t, 5, summary.MonthlyUsage)
	})

	t.Run("quota exceeded without credits", func(t *testing.T) {
		repo := &mockBillingRepo{
			billing: &billingdomain.OrgBilling{
				OrgID:            orgID,
				Plan:             billingdomain.PlanFree,
				PlanStatus:       "active",
				InterviewCredits: 0,
			},
			usage: 10,
		}
		svc := billingapp.NewBillingService(nil, repo, gw)
		allowed, err := svc.CheckAndConsumeQuota(context.Background(), orgID)
		require.Error(t, err)
		require.False(t, allowed)
	})

	t.Run("quota exceeded with credits succeeds", func(t *testing.T) {
		repo := &mockBillingRepo{
			billing: &billingdomain.OrgBilling{
				OrgID:            orgID,
				Plan:             billingdomain.PlanFree,
				PlanStatus:       "active",
				InterviewCredits: 2,
			},
			usage: 10,
		}
		svc := billingapp.NewBillingService(nil, repo, gw)
		allowed, err := svc.CheckAndConsumeQuota(context.Background(), orgID)
		require.NoError(t, err)
		require.True(t, allowed)
		require.Equal(t, 1, repo.billing.InterviewCredits)
	})

	t.Run("plan limit calculations", func(t *testing.T) {
		require.Equal(t, 10, billingdomain.PlanFree.MonthlyInterviewLimit())
		require.Equal(t, 100, billingdomain.PlanStarter.MonthlyInterviewLimit())
		require.Equal(t, 1000, billingdomain.PlanPro.MonthlyInterviewLimit())
	})

	t.Run("checkout and portal session generation", func(t *testing.T) {
		repo := &mockBillingRepo{
			billing: &billingdomain.OrgBilling{
				OrgID:            orgID,
				Plan:             billingdomain.PlanStarter,
				StripeCustomerID: "cus_12345",
			},
		}
		svc := billingapp.NewBillingService(nil, repo, gw)
		session, err := svc.CreateCheckout(context.Background(), orgID, billingdomain.PlanPro, 0, "https://example.com/success", "https://example.com/cancel")
		require.NoError(t, err)
		require.NotEmpty(t, session.URL)

		_, err = svc.CreateCheckout(context.Background(), orgID, "invalid_tier", 0, "https://example.com/success", "https://example.com/cancel")
		require.Error(t, err)
		var de *sharederr.DomainError
		require.ErrorAs(t, err, &de)
		require.Equal(t, "INVALID_PLAN", de.Code)
	})
}

package application

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	billingdomain "github.com/intivai/backend/internal/billing/domain"
	sharederr "github.com/intivai/backend/internal/shared/errors"
	"github.com/intivai/backend/pkg/db"
	"gorm.io/gorm"
)

type BillingSummaryResult struct {
	Plan               string     `json:"plan"`
	PlanStatus         string     `json:"plan_status"`
	MonthlyLimit       int        `json:"monthly_limit"`
	MonthlyUsage       int        `json:"monthly_usage"`
	InterviewCredits   int        `json:"interview_credits"`
	CurrentPeriodStart *time.Time `json:"current_period_start,omitempty"`
	CurrentPeriodEnd   *time.Time `json:"current_period_end,omitempty"`
	HasCustomerAccount bool       `json:"has_customer_account"`
}

type BillingService struct {
	pool    *gorm.DB
	repo    billingdomain.BillingRepository
	gateway billingdomain.PaymentGateway
}

func NewBillingService(pool *gorm.DB, repo billingdomain.BillingRepository, gateway billingdomain.PaymentGateway) *BillingService {
	return &BillingService{pool: pool, repo: repo, gateway: gateway}
}

func (s *BillingService) runInTx(ctx context.Context, orgID string, fn func(tctx context.Context) error) error {
	if s.pool == nil {
		return fn(ctx)
	}
	return db.RunInTx(ctx, s.pool, orgID, fn)
}

// CheckAndConsumeQuota evaluates interview limits against subscription tier and one-time credits.
func (s *BillingService) CheckAndConsumeQuota(ctx context.Context, orgID uuid.UUID) (bool, error) {
	var allowed bool
	err := s.runInTx(ctx, orgID.String(), func(tctx context.Context) error {
		billing, err := s.repo.GetOrgBilling(tctx, orgID)
		if err != nil {
			return err
		}

		limit := billing.Plan.MonthlyInterviewLimit()
		usage, err := s.repo.GetMonthlyUsage(tctx, orgID, billing.CurrentPeriodStart)
		if err != nil {
			return err
		}

		if usage < limit {
			allowed = true
			return nil
		}

		// Tier quota exceeded — attempt credit deduction
		if billing.InterviewCredits > 0 {
			deducted, err := s.repo.DeductCredit(tctx, orgID)
			if err != nil {
				return err
			}
			if deducted {
				allowed = true
				return nil
			}
		}

		return sharederr.NewDomainError(
			"QUOTA_EXCEEDED",
			fmt.Sprintf("monthly quota of %d interviews reached for plan %s; upgrade plan or purchase credits", limit, billing.Plan),
		)
	})
	return allowed, err
}

func (s *BillingService) GetSummary(ctx context.Context, orgID uuid.UUID) (*BillingSummaryResult, error) {
	var out *BillingSummaryResult
	err := s.runInTx(ctx, orgID.String(), func(tctx context.Context) error {
		b, err := s.repo.GetOrgBilling(tctx, orgID)
		if err != nil {
			return err
		}
		usage, err := s.repo.GetMonthlyUsage(tctx, orgID, b.CurrentPeriodStart)
		if err != nil {
			return err
		}
		out = &BillingSummaryResult{
			Plan:               string(b.Plan),
			PlanStatus:         b.PlanStatus,
			MonthlyLimit:       b.Plan.MonthlyInterviewLimit(),
			MonthlyUsage:       usage,
			InterviewCredits:   b.InterviewCredits,
			CurrentPeriodStart: b.CurrentPeriodStart,
			CurrentPeriodEnd:   b.CurrentPeriodEnd,
			HasCustomerAccount: b.StripeCustomerID != "",
		}
		return nil
	})
	return out, err
}

func (s *BillingService) CreateCheckout(ctx context.Context, orgID uuid.UUID, plan billingdomain.Plan, credits int, successURL, cancelURL string) (*billingdomain.CheckoutSession, error) {
	if !plan.IsValid() {
		return nil, sharederr.NewDomainError("INVALID_PLAN", "unknown billing plan")
	}
	return s.gateway.CreateCheckoutSession(ctx, billingdomain.CreateCheckoutCommand{
		OrgID:      orgID,
		Plan:       plan,
		Credits:    credits,
		SuccessURL: successURL,
		CancelURL:  cancelURL,
	})
}

func (s *BillingService) CreatePortal(ctx context.Context, orgID uuid.UUID, returnURL string) (string, error) {
	var customerID string
	err := db.RunInTx(ctx, s.pool, orgID.String(), func(tctx context.Context) error {
		b, err := s.repo.GetOrgBilling(tctx, orgID)
		if err != nil {
			return err
		}
		customerID = b.StripeCustomerID
		return nil
	})
	if err != nil {
		return "", err
	}
	if customerID == "" {
		return "", sharederr.NewDomainError("NO_CUSTOMER", "no active billing customer found for this organization")
	}
	return s.gateway.CreatePortalSession(ctx, customerID, returnURL)
}

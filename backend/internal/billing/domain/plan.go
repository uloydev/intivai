package domain

import (
	"time"

	"github.com/google/uuid"
)

type Plan string

const (
	PlanFree       Plan = "free"
	PlanStarter    Plan = "starter"
	PlanPro        Plan = "pro"
	PlanEnterprise Plan = "enterprise"
)

func (p Plan) MonthlyInterviewLimit() int {
	switch p {
	case PlanStarter:
		return 100
	case PlanPro:
		return 1000
	case PlanEnterprise:
		return 1000000 // custom / unlimited
	default:
		return 10 // Free tier default (docs/product/pricing.md)
	}
}

func (p Plan) IsValid() bool {
	switch p {
	case PlanFree, PlanStarter, PlanPro, PlanEnterprise:
		return true
	default:
		return false
	}
}

type OrgBilling struct {
	OrgID                uuid.UUID  `json:"org_id"`
	Plan                 Plan       `json:"plan"`
	PlanStatus           string     `json:"plan_status"` // "active", "past_due", "canceled"
	StripeCustomerID     string     `json:"stripe_customer_id,omitempty"`
	StripeSubscriptionID string     `json:"stripe_subscription_id,omitempty"`
	CurrentPeriodStart   *time.Time `json:"current_period_start,omitempty"`
	CurrentPeriodEnd     *time.Time `json:"current_period_end,omitempty"`
	InterviewCredits     int        `json:"interview_credits"`
}

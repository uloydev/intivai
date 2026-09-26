package persistence

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	billingdomain "github.com/intivai/backend/internal/billing/domain"
	"github.com/intivai/backend/pkg/db"
	"gorm.io/gorm"
)

type PostgresBillingRepo struct {
	pool *gorm.DB
}

func NewPostgresBillingRepo(pool *gorm.DB) *PostgresBillingRepo {
	return &PostgresBillingRepo{pool: pool}
}

func (r *PostgresBillingRepo) q(ctx context.Context) (*gorm.DB, error) {
	tx, ok := db.TxFrom(ctx)
	if !ok {
		return nil, db.ErrNoTx
	}
	return tx, nil
}

func (r *PostgresBillingRepo) GetOrgBilling(ctx context.Context, orgID uuid.UUID) (*billingdomain.OrgBilling, error) {
	q, err := r.q(ctx)
	if err != nil {
		return nil, err
	}
	row := q.WithContext(ctx).Raw(
		`SELECT id, COALESCE(plan, 'free'), COALESCE(plan_status, 'active'),
		        COALESCE(stripe_customer_id, ''), COALESCE(stripe_subscription_id, ''),
		        current_period_start, current_period_end, COALESCE(interview_credits, 0)
		 FROM orgs WHERE id = $1`, orgID).Row()

	var (
		b       billingdomain.OrgBilling
		planStr string
	)
	if err := row.Scan(&b.OrgID, &planStr, &b.PlanStatus, &b.StripeCustomerID, &b.StripeSubscriptionID,
		&b.CurrentPeriodStart, &b.CurrentPeriodEnd, &b.InterviewCredits); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &billingdomain.OrgBilling{OrgID: orgID, Plan: billingdomain.PlanFree, PlanStatus: "active"}, nil
		}
		return nil, err
	}
	b.Plan = billingdomain.Plan(planStr)
	return &b, nil
}

func (r *PostgresBillingRepo) UpdateSubscription(ctx context.Context, billing *billingdomain.OrgBilling) error {
	q, err := r.q(ctx)
	if err != nil {
		return err
	}
	return q.WithContext(ctx).Exec(
		`UPDATE orgs
		 SET plan = $1, plan_status = $2, stripe_customer_id = $3, stripe_subscription_id = $4,
		     current_period_start = $5, current_period_end = $6, interview_credits = $7,
		     updated_at = NOW()
		 WHERE id = $8`,
		string(billing.Plan), billing.PlanStatus, billing.StripeCustomerID, billing.StripeSubscriptionID,
		billing.CurrentPeriodStart, billing.CurrentPeriodEnd, billing.InterviewCredits, billing.OrgID).Error
}

func (r *PostgresBillingRepo) GetMonthlyUsage(ctx context.Context, orgID uuid.UUID, since *time.Time) (int, error) {
	q, err := r.q(ctx)
	if err != nil {
		return 0, err
	}
	var count int64
	err = q.WithContext(ctx).Raw(
		`SELECT COUNT(i.id)
		 FROM interviews i
		 JOIN applications a ON a.id = i.application_id
		 WHERE a.org_id = $1 AND ($2::timestamptz IS NULL OR i.created_at >= $2)`,
		orgID, since).Scan(&count).Error
	return int(count), err
}

func (r *PostgresBillingRepo) DeductCredit(ctx context.Context, orgID uuid.UUID) (bool, error) {
	q, err := r.q(ctx)
	if err != nil {
		return false, err
	}
	res := q.WithContext(ctx).Exec(
		`UPDATE orgs
		 SET interview_credits = interview_credits - 1, updated_at = NOW()
		 WHERE id = $1 AND interview_credits > 0`, orgID)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

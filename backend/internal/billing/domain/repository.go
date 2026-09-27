package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type BillingRepository interface {
	GetOrgBilling(ctx context.Context, orgID uuid.UUID) (*OrgBilling, error)
	UpdateSubscription(ctx context.Context, billing *OrgBilling) error
	GetMonthlyUsage(ctx context.Context, orgID uuid.UUID, since *time.Time) (int, error)
	DeductCredit(ctx context.Context, orgID uuid.UUID) (bool, error)
}

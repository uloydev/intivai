package application

import (
	"context"

	"github.com/google/uuid"
)

// TxManager — driven port for transactional use cases (e.g. org registration
// where RLS tenant must be set BEFORE the first insert).
type TxManager interface {
	// RunInTx runs fn inside a transaction. When tenantID is non-nil the
	// transaction sets app.org_id = tenantID first, so RLS policies apply.
	RunInTx(ctx context.Context, tenantID *uuid.UUID, fn func(ctx context.Context) error) error
}

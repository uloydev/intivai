package persistence

import (
	"context"

	"github.com/google/uuid"
	"github.com/intivai/backend/pkg/db"
	"gorm.io/gorm"
)

// PostgresTxManager implements application.TxManager with a GORM handle.
// When tenantID is set, app.org_id is configured on the transaction BEFORE
// any statement so RLS policies resolve correctly.
type PostgresTxManager struct {
	pool *gorm.DB
}

func NewPostgresTxManager(pool *gorm.DB) *PostgresTxManager {
	return &PostgresTxManager{pool: pool}
}

func (m *PostgresTxManager) RunInTx(ctx context.Context, tenantID *uuid.UUID, fn func(ctx context.Context) error) error {
	return m.pool.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if tenantID != nil {
			if err := db.SetTenant(ctx, tx, tenantID.String()); err != nil {
				return err
			}
		}
		return fn(db.WithTx(ctx, tx))
	})
}

package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // register "pgx" database/sql driver
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// NewPool opens a GORM handle backed by a pgx stdlib connection pool.
func NewPool(ctx context.Context, url string) (*gorm.DB, error) {
	sqlDB, err := sql.Open("pgx", url)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	sqlDB.SetMaxOpenConns(20)
	sqlDB.SetMaxIdleConns(2)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)
	sqlDB.SetConnMaxIdleTime(5 * time.Minute)

	gdb, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{})
	if err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("gorm open: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(pingCtx); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return gdb, nil
}

// ---- Tenant + transaction context ----

type tenantKey struct{}
type txKey struct{}

// WithTenant attaches the tenant (org_id) to a context. Actual RLS resolution
// requires SetTenant on the SAME connection/transaction — see SetTenant.
func WithTenant(ctx context.Context, orgID string) context.Context {
	return context.WithValue(ctx, tenantKey{}, orgID)
}

func TenantFrom(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(tenantKey{}).(string)
	return v, ok && v != ""
}

// WithTx attaches a transaction (*gorm.DB) to the context. Repos resolve it
// via TxFrom.
func WithTx(ctx context.Context, tx *gorm.DB) context.Context {
	return context.WithValue(ctx, txKey{}, tx)
}

func TxFrom(ctx context.Context) (*gorm.DB, bool) {
	tx, ok := ctx.Value(txKey{}).(*gorm.DB)
	return tx, ok && tx != nil
}

var (
	ErrNoTenant = errors.New("no tenant in context")
	ErrNoTx     = errors.New("no transaction in context (tenant tables require one)")
)

// SetTenant runs SELECT set_config('app.org_id', ...) on the given gorm handle
// (pool or transaction). MUST be called inside a transaction before any
// RLS-scoped statement.
func SetTenant(ctx context.Context, q *gorm.DB, orgID string) error {
	if orgID == "" {
		return ErrNoTenant
	}
	return q.WithContext(ctx).Exec("SELECT set_config('app.org_id', $1, true)", orgID).Error
}

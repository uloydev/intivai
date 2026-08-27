package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // register "pgx" database/sql driver
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// txTracer resolves the global provider PER CALL — a package-level
// otel.Tracer var freezes onto the first installed provider (global
// delegation is once-only) and silently misses later Inits.
func txTracer() trace.Tracer {
	return otel.Tracer("github.com/intivai/backend/pkg/db")
}

// PoolOption customizes pool construction. Plugins (e.g. otelgorm) are
// injected by the caller so this package stays free of observability deps.
type PoolOption func(*poolOptions)

type poolOptions struct {
	plugins []gorm.Plugin
}

// WithPlugin registers a gorm plugin on the opened handle. Registration
// failures abort NewPool (a broken tracing plugin must not boot half-wired).
func WithPlugin(p gorm.Plugin) PoolOption {
	return func(o *poolOptions) { o.plugins = append(o.plugins, p) }
}

// NewPool opens a GORM handle backed by a pgx stdlib connection pool.
func NewPool(ctx context.Context, url string, opts ...PoolOption) (*gorm.DB, error) {
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
		_ = sqlDB.Close()
		return nil, fmt.Errorf("gorm open: %w", err)
	}

	po := &poolOptions{}
	for _, opt := range opts {
		opt(po)
	}
	for _, p := range po.plugins {
		if err := gdb.Use(p); err != nil {
			_ = sqlDB.Close()
			return nil, fmt.Errorf("gorm plugin %T: %w", p, err)
		}
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(pingCtx); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return gdb, nil
}

// ---- Tenant + transaction context ----

type tenantKey struct{}
type txKey struct{}
type afterCommitKey struct{}

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

// AfterCommit registers fn to run AFTER the surrounding transaction commits.
// Fire-and-forget side effects (async enqueues, emails) depend on the row
// being durable: enqueuing inside the tx races the commit (a worker can read
// the row before it commits) and an enqueue error mid-request rolls the
// business update back while the task is already queued (J8). Registered
// hooks run in registration order from RunAfterCommit, which the tenant-tx
// middleware calls right after a successful commit. Hooks must be
// non-blocking-friendly: they run synchronously on the request goroutine and
// their errors are logged by the caller, never propagated to the handler.
func AfterCommit(ctx context.Context, fn func(ctx context.Context)) {
	if fn == nil {
		return
	}
	v, ok := ctx.Value(afterCommitKey{}).(*afterCommitHooks)
	if !ok {
		// No transaction envelope in this context (worker path): the caller is
		// outside RunInTx semantics — run inline so the side effect is not lost.
		fn(ctx)
		return
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	v.fns = append(v.fns, fn)
}

type afterCommitHooks struct {
	mu  sync.Mutex
	fns []func(ctx context.Context)
}

// RunAfterCommit executes every hook registered via AfterCommit on ctx, in
// registration order. Intended to be invoked once, immediately after the
// enclosing transaction commits. Callers (tenant-tx middleware, RunInTx) log
// hook errors; a failing hook must not roll back an already-committed
// transaction.
func RunAfterCommit(ctx context.Context) {
	v, ok := ctx.Value(afterCommitKey{}).(*afterCommitHooks)
	if !ok {
		return
	}
	v.mu.Lock()
	fns := v.fns
	v.fns = nil
	v.mu.Unlock()
	for _, fn := range fns {
		fn(ctx)
	}
}

// WithAfterCommit attaches a transient hook registry to ctx. RunInTx uses it
// internally so hooks registered inside a transaction body fire once after
// commit.
func WithAfterCommit(ctx context.Context) context.Context {
	return context.WithValue(ctx, afterCommitKey{}, &afterCommitHooks{})
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

// RunInTx executes fn inside a transaction with app.org_id set, so RLS
// policies apply. Workers and handlers outside the tenant-tx middleware use
// this to wrap repo calls. Commits on success, rolls back on error. If the
// context already carries a request transaction (tenant-tx middleware), it is
// reused — opening a second pool connection per request deadlocks the pool
// under concurrency (N goroutines × 2 conns vs MaxOpenConns).
func RunInTx(ctx context.Context, pool *gorm.DB, orgID string, fn func(ctx context.Context) error) error {
	// tenant.tx span: org.id attr (UUID only — no PII), SQL children nest
	// under it via the otelgorm plugin (plan §5).
	ctx, span := txTracer().Start(ctx, "tenant.tx", trace.WithAttributes(attribute.String("org.id", orgID)))
	defer span.End()

	if _, ok := TxFrom(ctx); ok {
		err := fn(ctx)
		recordTxError(span, err)
		return err
	}
	// Standalone transaction (worker/service path): attach a hook registry so
	// AfterCommit side effects fire only after this tx commits (J8). When a
	// request tx exists, the registry lives on the request ctx and the
	// tenant-tx middleware fires it after the request commit instead.
	ctx = WithAfterCommit(ctx)
	err := pool.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := SetTenant(ctx, tx, orgID); err != nil {
			return err
		}
		return fn(WithTx(ctx, tx))
	})
	recordTxError(span, err)
	if err == nil {
		RunAfterCommit(ctx)
	}
	return err
}

func recordTxError(span trace.Span, err error) {
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
}

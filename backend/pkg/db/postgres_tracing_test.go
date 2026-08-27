package db

import (
	"context"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/uptrace/opentelemetry-go-extra/otelgorm"
	"gorm.io/gorm"

	"github.com/intivai/backend/pkg/telemetry"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type memExporter struct {
	mu    sync.Mutex
	spans []sdktrace.ReadOnlySpan
}

func (m *memExporter) ExportSpans(_ context.Context, spans []sdktrace.ReadOnlySpan) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.spans = append(m.spans, spans...)
	return nil
}

func (m *memExporter) Shutdown(context.Context) error { return nil }

// A RunInTx call must appear as a `tenant.tx` span carrying org.id, with the
// SQL work nested as child spans (plan §5 matrix). This is what makes an
// I19-class failure visible as an errored db span instead of a bare 500.
func TestRunInTxProducesTenantAndDBSpans(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	exp := &memExporter{}
	shutdown, err := telemetry.Init(context.Background(), telemetry.Config{
		Enable: true, ServiceName: "db-tracing-test", Env: "dev", SampleRatio: 1,
	}, telemetry.WithExporter(exp))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = shutdown(context.Background()) })

	pool, err := NewPool(context.Background(), url, WithPlugin(otelgorm.NewPlugin()))
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := pool.DB()
	t.Cleanup(func() { _ = sqlDB.Close() })

	// Fresh org id: set_config accepts any string and SELECT 1 touches no
	// RLS-scoped table, so this stays independent of seed data.
	orgID := uuid.NewString()
	err = RunInTx(context.Background(), pool, orgID, func(ctx context.Context) error {
		var one int
		return pool.WithContext(ctx).Raw("SELECT 1").Row().Scan(&one)
	})
	if err != nil {
		t.Fatalf("probe tx: %v", err)
	}

	var tenantSpan sdktrace.ReadOnlySpan
	dbChildCount := 0
	for _, s := range exp.spans {
		if s.Name() == "tenant.tx" {
			tenantSpan = s
			continue
		}
		if s.Parent().IsValid() {
			dbChildCount++
		}
	}
	if tenantSpan == nil {
		names := make([]string, 0, len(exp.spans))
		for _, s := range exp.spans {
			names = append(names, s.Name())
		}
		t.Fatalf("tenant.tx span missing; exported: %v", names)
	}
	foundOrg := false
	for _, attr := range tenantSpan.Attributes() {
		if attr.Key == "org.id" && attr.Value.AsString() == orgID {
			foundOrg = true
		}
	}
	if !foundOrg {
		t.Fatal("org.id attribute missing from tenant.tx span")
	}
	if dbChildCount < 1 {
		t.Fatalf("expected >=1 db child span under tenant.tx, got %d", dbChildCount)
	}
}

// J16: repo-style calls resolve the tx via TxFrom(ctx) and run statements on
// that handle — the otelgorm spans must nest DIRECTLY under the tenant.tx
// span created by the same RunInTx call.
func TestRunInTxTxFromNestsDBSpansUnderTenantSpan(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	exp := &memExporter{}
	shutdown, err := telemetry.Init(context.Background(), telemetry.Config{
		Enable: true, ServiceName: "db-tracing-test", Env: "dev", SampleRatio: 1,
	}, telemetry.WithExporter(exp))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = shutdown(context.Background()) })

	pool, err := NewPool(context.Background(), url, WithPlugin(otelgorm.NewPlugin()))
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := pool.DB()
	t.Cleanup(func() { _ = sqlDB.Close() })

	orgID := uuid.NewString()
	err = RunInTx(context.Background(), pool, orgID, func(ctx context.Context) error {
		tx, ok := TxFrom(ctx)
		if !ok {
			return ErrNoTx
		}
		var one int
		return tx.WithContext(ctx).Raw("SELECT 1").Row().Scan(&one)
	})
	if err != nil {
		t.Fatalf("probe tx: %v", err)
	}

	var tenantSpanID string
	for _, s := range exp.spans {
		if s.Name() == "tenant.tx" {
			tenantSpanID = s.SpanContext().SpanID().String()
			break
		}
	}
	if tenantSpanID == "" {
		t.Fatal("tenant.tx span missing")
	}
	dbChildCount := 0
	for _, s := range exp.spans {
		if s.Name() == "tenant.tx" {
			continue
		}
		if p := s.Parent(); p.SpanID().String() == tenantSpanID && p.TraceID().String() != "" {
			dbChildCount++
		}
	}
	if dbChildCount == 0 {
		t.Fatal("no db span parented under tenant.tx — TxFrom handle not traced under the tenant span")
	}
}

// J16 reuse case: RunInTx called with a context that already carries a
// transaction (tenant-tx middleware path) must reuse it — no nested second
// transaction (which would open a second pool connection, deadlocking under
// concurrency) and no second SetTenant. The context sees the SAME *gorm.DB
// handle, and the single tenant.tx span roots all db spans.
func TestRunInTxReusesOuterTransaction(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	exp := &memExporter{}
	shutdown, err := telemetry.Init(context.Background(), telemetry.Config{
		Enable: true, ServiceName: "db-tracing-test", Env: "dev", SampleRatio: 1,
	}, telemetry.WithExporter(exp))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = shutdown(context.Background()) })

	pool, err := NewPool(context.Background(), url, WithPlugin(otelgorm.NewPlugin()))
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := pool.DB()
	t.Cleanup(func() { _ = sqlDB.Close() })

	orgID := uuid.NewString()
	err = pool.WithContext(context.Background()).Transaction(func(tx *gorm.DB) error {
		// tenant-tx middleware does exactly this: WithTx(userCtx, tx)
		return RunInTx(WithTx(context.Background(), tx), pool, orgID, func(inner context.Context) error {
			itx, ok := TxFrom(inner)
			if !ok {
				return ErrNoTx
			}
			// Strongest proof of reuse: the identical handle.
			if itx != tx {
				t.Error("RunInTx did not reuse the outer transaction handle")
			}
			var one int
			return itx.WithContext(inner).Raw("SELECT 1").Row().Scan(&one)
		})
	})
	if err != nil {
		t.Fatalf("reuse tx: %v", err)
	}

	tenantCount := 0
	var tenantSpanID string
	for _, s := range exp.spans {
		if s.Name() == "tenant.tx" {
			tenantCount++
			tenantSpanID = s.SpanContext().SpanID().String()
		}
	}
	if tenantCount != 1 {
		t.Fatalf("tenant.tx spans: %d want 1", tenantCount)
	}
	// SetTenant runs only in the create-tx path; a nested second transaction
	// would show up as an extra set_config statement.
	var setConfigCount int
	var dbChildCount int
	for _, s := range exp.spans {
		if s.Name() != "tenant.tx" {
			if p := s.Parent(); p.SpanID().String() == tenantSpanID && p.TraceID().String() != "" {
				dbChildCount++
			}
		}
		for _, attr := range s.Attributes() {
			if attr.Key == "db.statement" && strings.Contains(attr.Value.AsString(), "set_config") {
				setConfigCount++
			}
		}
	}
	if setConfigCount != 0 {
		t.Fatalf("set_config executed %d times: nested transaction created", setConfigCount)
	}
	if dbChildCount == 0 {
		t.Fatal("no db span parented under the reused tenant.tx span")
	}
}

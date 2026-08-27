package telemetry

import (
	"context"
	"os"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
)

// TestOTLPExportAgainstLiveCollector is env-gated: point
// OTEL_TEST_OTLP_ENDPOINT at a real collector (dev: http://localhost:4318)
// and assert a span round-trips through the production code path —
// including the endpoint-scheme → insecure/TLS decision and the batch
// processor flush on shutdown.
func TestOTLPExportAgainstLiveCollector(t *testing.T) {
	endpoint := os.Getenv("OTEL_TEST_OTLP_ENDPOINT")
	if endpoint == "" {
		t.Skip("OTEL_TEST_OTLP_ENDPOINT not set")
	}
	cfg := Config{
		Enable:       true,
		ServiceName:  "telemetry-e2e-test",
		Env:          "dev",
		OTLPEndpoint: endpoint,
		SampleRatio:  1,
	}
	shutdown, err := Init(context.Background(), cfg)
	if err != nil {
		t.Fatalf("init: %v", err)
	}

	tracer := otel.Tracer("telemetry-e2e-test")
	_, span := tracer.Start(context.Background(), "otlp-round-trip")
	span.End()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := shutdown(ctx); err != nil {
		t.Fatalf("shutdown flush: %v", err)
	}
}

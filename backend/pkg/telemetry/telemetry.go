// Package telemetry owns OpenTelemetry tracing bootstrap: one place to turn
// spans on, one place to shut them down. Disabled mode is a hard noop —
// nothing in this package may degrade API behavior when the collector is
// absent (plan docs/plans/active/otel-tracing-plan-2026-08-26.md, D8).
package telemetry

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

const shutdownTimeout = 5 * time.Second

// droppedSpans counts failed export attempts (collector down, export
// timeout, exporter errors) — the SDK routes those through the global
// ErrorHandler. Queue-full enqueue drops have no SDK callback; they are
// visible in the SDK's own observability metrics only.
var droppedSpans atomic.Int64

// Config mirrors pkg/config.Telemetry; kept separate so telemetry has no
// dependency on the config package (and tests need no viper).
type Config struct {
	Enable       bool
	ServiceName  string  // resource service.name (e.g. intivai-server)
	Env          string  // deployment environment (dev | prod)
	OTLPEndpoint string  // e.g. http://jaeger:4318 or jaeger:4318
	SampleRatio  float64 // (0,1] — head sampling via ParentBased(TraceIDRatioBased)
}

type options struct {
	exporter sdktrace.SpanExporter // test seam: replaces OTLP exporter + enables sync processing
	batch    bool                  // test seam: route exporter behind the production batch processor
}

// Option customizes Init. WithExporter exists for deterministic tests:
// a custom exporter runs behind a SimpleSpanProcessor so exported spans are
// observable without polling.
type Option func(*options)

func WithExporter(e sdktrace.SpanExporter) Option {
	return func(o *options) { o.exporter = e }
}

// WithBatchExporter runs the exporter behind the production BatchSpanProcessor
// configuration (same queue/batch/timeout bounds as OTLP) — test seam for
// collector-down and queue-full behavior: the real path must never let a
// stuck exporter stall span.End().
func WithBatchExporter(e sdktrace.SpanExporter) Option {
	return func(o *options) { o.exporter = e; o.batch = true }
}

// ShutdownFunc flushes and stops the provider. Safe to call multiple times.
type ShutdownFunc func(context.Context) error

// Init wires the global tracer provider + propagators. When cfg.Enable is
// false it returns a noop-safe shutdown and leaves globals untouched.
// Every enabled Init installs the error handler that feeds DroppedSpanCount
// (replacing OTel's default log-only handler — same logging, plus the count).
func Init(_ context.Context, cfg Config, opts ...Option) (ShutdownFunc, error) {
	o := &options{}
	for _, opt := range opts {
		opt(o)
	}

	if !cfg.Enable {
		return func(context.Context) error { return nil }, nil
	}
	if cfg.ServiceName == "" {
		return nil, errors.New("telemetry: service_name required when enabled")
	}
	if cfg.SampleRatio <= 0 || cfg.SampleRatio > 1 {
		return nil, fmt.Errorf("telemetry: sample_ratio must be in (0,1], got %v", cfg.SampleRatio)
	}

	// Export failures (exporter errors, dropped batches, export timeouts) all
	// route through otel.Handle; count them so operators can see telemetry
	// silently losing data (J3). We canNOT chain the previous handler: once
	// SetErrorHandler installs a delegate, the default ErrDelegator forwards
	// to us — chaining would recurse. The SDK default is log.Print; we do the
	// same, but with the counter first so dropped spans are observable.
	droppedSpans.Store(0)
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		droppedSpans.Add(1)
		log.Printf("telemetry: span export dropped: %v", err)
	}))

	var spanProc sdktrace.SpanProcessor
	if o.exporter != nil {
		if o.batch {
			spanProc = sdktrace.NewBatchSpanProcessor(
				o.exporter,
				sdktrace.WithMaxQueueSize(4096),
				sdktrace.WithMaxExportBatchSize(256),
				sdktrace.WithExportTimeout(5*time.Second),
			)
		} else {
			spanProc = sdktrace.NewSimpleSpanProcessor(o.exporter)
		}
	} else {
		exp, err := newOTLPExporter(cfg.OTLPEndpoint)
		if err != nil {
			return nil, fmt.Errorf("telemetry: otlp exporter: %w", err)
		}
		spanProc = sdktrace.NewBatchSpanProcessor(
			exp,
			sdktrace.WithMaxQueueSize(4096),
			sdktrace.WithMaxExportBatchSize(256),
			// Bounded non-blocking export (J3): a full queue or a down
			// collector drops spans instead of stalling request/worker/WS
			// paths. ExportTimeout bounds each batch export attempt.
			sdktrace.WithExportTimeout(5*time.Second),
		)
	}

	res, err := resource.Merge(resource.Default(),
		resource.NewSchemaless(
			attribute.String("service.name", cfg.ServiceName),
			attribute.String("deployment.environment.name", cfg.Env),
		))
	if err != nil {
		return nil, fmt.Errorf("telemetry: resource: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(cfg.SampleRatio))),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(spanProc),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	shutdown := func(ctx context.Context) error { return tp.Shutdown(ctx) }
	return once(shutdown), nil
}

// DroppedSpanCount reports how many telemetry export attempts failed — a
// non-zero value means spans are being silently lost (collector down,
// export timeout, exporter error). Reset by each enabled Init.
func DroppedSpanCount() int64 { return droppedSpans.Load() }

// newOTLPExporter builds an OTLP/HTTP exporter. Endpoint rules:
// "https://host" → TLS; "http://host" or bare "host" → insecure (compose-internal traffic).
func newOTLPExporter(endpoint string) (sdktrace.SpanExporter, error) {
	e := strings.TrimPrefix(endpoint, "http://")
	insecure := e != endpoint || !strings.Contains(endpoint, "://")
	if https := strings.TrimPrefix(endpoint, "https://"); https != endpoint {
		e = https
		insecure = false
	}
	if e == "" {
		return nil, errors.New("telemetry: otlp endpoint empty")
	}
	opts := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(e),
		otlptracehttp.WithTimeout(5 * time.Second),
	}
	if insecure {
		opts = append(opts, otlptracehttp.WithInsecure())
	}
	return otlptracehttp.New(context.Background(), opts...)
}

// once makes shutdown idempotent — TracerProvider.Shutdown panics on second
// call, and the provider itself is only shutdown-once safe. sync.Once also
// guarantees exactly-one call under concurrent shutdown (J14).
func once(fn ShutdownFunc) ShutdownFunc {
	var do sync.Once
	return func(ctx context.Context) error {
		var err error
		do.Do(func() {
			sctx, cancel := context.WithTimeout(ctx, shutdownTimeout)
			defer cancel()
			err = fn(sctx)
			otel.SetTracerProvider(noopTracerProvider())
		})
		return err
	}
}

func noopTracerProvider() trace.TracerProvider {
	return noop.NewTracerProvider()
}

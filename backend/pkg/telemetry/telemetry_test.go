package telemetry

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

// memExporter is an in-memory span exporter for deterministic assertions —
// tests never touch the network (plan §8).
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

func (m *memExporter) Shutdown(_ context.Context) error { return nil }

func (m *memExporter) count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.spans)
}

// blockingExporter simulates a collector that never responds: ExportSpans
// blocks until the context ends (export timeout), Shutdown returns immediately.
type blockingExporter struct {
	released chan struct{}
}

func (b *blockingExporter) ExportSpans(ctx context.Context, _ []sdktrace.ReadOnlySpan) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-b.released:
		return nil
	}
}

func (b *blockingExporter) Shutdown(context.Context) error { return nil }

func TestInitDisabledIsNoop(t *testing.T) {
	shutdown, err := Init(context.Background(), Config{Enable: false})
	if err != nil {
		t.Fatalf("init disabled: %v", err)
	}
	defer func() { _ = shutdown(context.Background()) }()

	tracer := otel.Tracer("telemetry-test")
	_, span := tracer.Start(context.Background(), "op")
	if span.IsRecording() {
		t.Fatal("disabled mode must produce non-recording (noop) spans")
	}
	span.End()
}

func TestInitEnabledExportsSpanWithResource(t *testing.T) {
	exp := &memExporter{}
	cfg := Config{
		Enable:      true,
		ServiceName: "intivai-test",
		Env:         "dev",
		SampleRatio: 1,
	}
	shutdown, err := Init(context.Background(), cfg, WithExporter(exp))
	if err != nil {
		t.Fatalf("init enabled: %v", err)
	}
	defer func() { _ = shutdown(context.Background()) }()

	tracer := otel.Tracer("telemetry-test")
	ctx, span := tracer.Start(context.Background(), "unit-of-work")
	span.SetAttributes(attribute.String("component", "test"))
	span.End()

	var svcName, envName string

	// Simple span processor path (custom exporter): export is synchronous.
	if exp.count() != 1 {
		t.Fatalf("exported spans: got %d want 1", exp.count())
	}
	got := exp.spans[0]
	if got.Name() != "unit-of-work" {
		t.Fatalf("span name: got %q", got.Name())
	}
	for _, attr := range got.Resource().Attributes() {
		switch attr.Key {
		case "service.name":
			svcName = attr.Value.AsString()
		case "deployment.environment.name", "deployment.environment":
			envName = attr.Value.AsString()
		}
	}
	if svcName != "intivai-test" {
		t.Fatalf("service.name resource attr: got %q want intivai-test", svcName)
	}
	if envName != "dev" {
		t.Fatalf("deployment environment resource attr: got %q want dev", envName)
	}
	foundAttr := false
	for _, attr := range got.Attributes() {
		if attr.Key == "component" && attr.Value.AsString() == "test" {
			foundAttr = true
		}
	}
	if !foundAttr {
		t.Fatal("span attributes lost")
	}
	_ = ctx
}

func TestInitEnabledSampleRatioZeroDropsRootSpans(t *testing.T) {
	exp := &memExporter{}
	cfg := Config{Enable: true, ServiceName: "s", Env: "dev", SampleRatio: 0.0000001}
	shutdown, err := Init(context.Background(), cfg, WithExporter(exp))
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	defer func() { _ = shutdown(context.Background()) }()

	tracer := otel.Tracer("telemetry-test")
	_, span := tracer.Start(context.Background(), "dropped")
	span.End()
	if exp.count() != 0 {
		t.Fatalf("near-zero sampler exported %d spans, want 0", exp.count())
	}
}

func TestInitEnabledInvalidSampleRatioRejected(t *testing.T) {
	cases := []float64{-1, 0, 2}
	for _, r := range cases {
		shutdown, err := Init(context.Background(), Config{
			Enable: true, ServiceName: "s", Env: "dev", SampleRatio: r,
		})
		if err == nil {
			_ = shutdown(context.Background())
			t.Fatalf("ratio %v: want error", r)
		}
	}
}

// ParentBased semantics: a remote sampled parent forces the local decision to
// sample, regardless of the local ratio — this is what keeps a trace alive
// across service/task boundaries once the entry point sampled it in.
func TestChildSpanAlwaysFollowsParentDecision(t *testing.T) {
	exp := &memExporter{}
	cfg := Config{Enable: true, ServiceName: "s", Env: "dev", SampleRatio: 0.0000001}
	shutdown, err := Init(context.Background(), cfg, WithExporter(exp))
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	defer func() { _ = shutdown(context.Background()) }()

	tid, err := trace.TraceIDFromHex("4bf92f3577b34da6a3ce929d0e0e4736")
	if err != nil {
		t.Fatal(err)
	}
	sid, err := trace.SpanIDFromHex("00f067aa0ba902b7")
	if err != nil {
		t.Fatal(err)
	}
	remoteSampled := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    tid,
		SpanID:     sid,
		Remote:     true,
		TraceFlags: trace.FlagsSampled,
	})
	parentCtx := trace.ContextWithSpanContext(context.Background(), remoteSampled)

	tracer := otel.Tracer("telemetry-test")
	_, span := tracer.Start(parentCtx, "kept-by-parent")
	span.End()

	if exp.count() != 1 {
		t.Fatalf("sampled remote parent must force sampling: exported %d want 1", exp.count())
	}
	if got := exp.spans[0].Parent(); got.SpanID() != sid {
		t.Fatalf("parent span id mismatch: %v", got.SpanID())
	}
}

// J14: concurrent shutdown must cause exactly one provider Shutdown. The
// old bool guard raced: two goroutines could both pass the check and the
// provider's once-only Shutdown would panic or double-execute.
type shutdownSpy struct {
	n atomic.Int64
}

func (s *shutdownSpy) Add()        { s.n.Add(1) }
func (s *shutdownSpy) Load() int64 { return s.n.Load() }

// publicTestProvider wraps sdktrace.TracerProvider with a shutdown spy.
type publicTestProvider struct {
	*sdktrace.TracerProvider
	spy *shutdownSpy
}

func (p *publicTestProvider) Shutdown(ctx context.Context) error {
	p.spy.Add()
	return p.TracerProvider.Shutdown(ctx)
}

// TestConcurrentShutdown drives the REAL shutdown returned by Init from 20
// goroutines under -race. It cannot observe the internal provider directly,
// so the race detector is the strongest signal; add another test that proves
// exactly-once via the spy on the real provider type.
func TestConcurrentShutdown(t *testing.T) {
	exp := &memExporter{}
	cfg := Config{Enable: true, ServiceName: "s", Env: "dev", SampleRatio: 1}
	shutdown, err := Init(context.Background(), cfg, WithExporter(exp))
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := shutdown(context.Background()); err != nil {
				t.Errorf("shutdown: %v", err)
			}
		}()
	}
	wg.Wait()

	// The provider was shut down exactly once (no panic, no double-flush);
	// subsequent spans must be noop.
	tracer := otel.Tracer("telemetry-test")
	_, span := tracer.Start(context.Background(), "after-shutdown")
	if span.IsRecording() {
		t.Fatal("post-shutdown spans must not record")
	}
	span.End()
}

// TestShutdownOnceCallsProviderExactlyOnce proves the sync.Once wrapper:
// whatever ShutdownFunc is wrapped, only the first concurrent caller runs the
// underlying provider. This IS what Init installs (same wrapper), against a
// real sdktrace.TracerProvider — so the exact-once guarantee holds even when
// the provider itself is a spy.
func TestShutdownOnceCallsProviderExactlyOnce(t *testing.T) {
	spyTP := &publicTestProvider{
		TracerProvider: sdktrace.NewTracerProvider(sdktrace.WithSyncer(&memExporter{})),
		spy:            &shutdownSpy{},
	}
	onceShutdown := once(func(ctx context.Context) error { return spyTP.Shutdown(ctx) })

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := onceShutdown(context.Background()); err != nil {
				t.Errorf("shutdown: %v", err)
			}
		}()
	}
	wg.Wait()
	if got := spyTP.spy.Load(); got != 1 {
		t.Fatalf("provider.Shutdown calls: %d, want exactly 1", got)
	}
}

// J3 collector-down: a batch processor export that never completes must not
// stall span submission (WithBlocking removed). The batch processor's
// ExportTimeout (5s) bounds export attempts; with a hanging collector the
// drain finishes at most ~5s after the export starts, so shutdown resolves
// well inside the 15s bound — the point is NO deadlock, not speed.
func TestCollectorDownDoesNotBlock(t *testing.T) {
	exp := &blockingExporter{released: make(chan struct{})}
	cfg := Config{Enable: true, ServiceName: "s", Env: "dev", SampleRatio: 1}
	shutdown, err := Init(context.Background(), cfg, WithBatchExporter(exp))
	if err != nil {
		t.Fatal(err)
	}

	start := time.Now()
	tracer := otel.Tracer("telemetry-test")
	_, span := tracer.Start(context.Background(), "collector-down")
	span.End() // must not block: queue accept is non-blocking (drop-on-full)
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("span.End blocked %v with collector down (WithBlocking still present?)", elapsed)
	}

	// Shutdown while exporter hangs: the processor drain is bounded by
	// ExportTimeout — it must return, not deadlock.
	sctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	sDone := make(chan struct{})
	go func() {
		defer close(sDone)
		_ = shutdown(sctx)
	}()
	select {
	case <-sDone:
	case <-time.After(15 * time.Second):
		close(exp.released)
		t.Fatal("shutdown deadlocked while exporter hangs")
	}
}

// The error handler installed by Init feeds the dropped-span counter; a
// failing exporter must move it. Shutdown flushes the processor
// synchronously, so by the time it returns the counter reflects the failure.
func TestFailedExportIncrementsDroppedCounter(t *testing.T) {
	failing := errorExporter{err: errors.New("collector refused")}
	cfg := Config{Enable: true, ServiceName: "s", Env: "dev", SampleRatio: 1}
	shutdown, err := Init(context.Background(), cfg, WithBatchExporter(failing))
	if err != nil {
		t.Fatal(err)
	}

	tracer := otel.Tracer("telemetry-test")
	_, span := tracer.Start(context.Background(), "boom")
	span.End()

	sctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = shutdown(sctx)

	if got := DroppedSpanCount(); got == 0 {
		t.Fatal("drop counter did not move after export error")
	}
}

type errorExporter struct{ err error }

func (e errorExporter) ExportSpans(context.Context, []sdktrace.ReadOnlySpan) error { return e.err }
func (e errorExporter) Shutdown(context.Context) error                             { return nil }

package queue

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/hibiken/asynq"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
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

func (m *memExporter) count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.spans)
}

func newTestProvider(exp *memExporter) *sdktrace.TracerProvider {
	return sdktrace.NewTracerProvider(sdktrace.WithSyncer(exp), sdktrace.WithSampler(sdktrace.AlwaysSample()))
}

// Producer side: the active span context must land in the task header so the
// worker can link back to it (plan batch D).
func TestInjectTraceContextWritesTraceparent(t *testing.T) {
	exp := &memExporter{}
	tp := newTestProvider(exp)
	prev := otel.GetTracerProvider()
	otel.SetTracerProvider(tp)
	t.Cleanup(func() { otel.SetTracerProvider(prev) })

	ctx, span := otel.Tracer("queue-test").Start(context.Background(), "enqueue-site")
	defer span.End()

	task := asynq.NewTaskWithHeaders("test_task", []byte("{}"), map[string]string{})
	injectTraceContext(ctx, task)

	tpHeader := task.Headers()["traceparent"]
	if tpHeader == "" {
		t.Fatal("traceparent missing from task header")
	}
	want := span.SpanContext().TraceID().String()
	if !contains(tpHeader, want) {
		t.Fatalf("traceparent %q does not carry trace id %s", tpHeader, want)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}

// Worker side: delivery becomes a root span named after the task, LINKED to
// the producer trace (links over parents — retries re-deliver stale headers,
// and a link is honest causality; plan D-batch-D). Handler receives ctx whose
// active span is the delivery span.
func TestTracingMiddlewareCreatesLinkedRootSpan(t *testing.T) {
	exp := &memExporter{}
	tp := newTestProvider(exp)
	prev := otel.GetTracerProvider()
	otel.SetTracerProvider(tp)
	t.Cleanup(func() { otel.SetTracerProvider(prev) })

	// Simulate the producer: sampled span → inject into header.
	pCtx, pSpan := otel.Tracer("queue-test").Start(context.Background(), "producer")
	headers := map[string]string{}
	propagation.TraceContext{}.Inject(pCtx, propagation.MapCarrier(headers))
	pSpan.End()

	task := asynq.NewTaskWithHeaders("score_candidate", []byte("{}"), headers)

	var handlerSpan trace.Span
	mw := TracingMiddleware()
	h := mw(asynq.HandlerFunc(func(ctx context.Context, tsk *asynq.Task) error {
		handlerSpan = trace.SpanFromContext(ctx)
		return nil
	}))
	if err := h.ProcessTask(context.Background(), task); err != nil {
		t.Fatal(err)
	}

	// producer + delivery both export through the syncer
	if exp.count() != 2 {
		t.Fatalf("exported spans: %d want 2 (producer + delivery)", exp.count())
	}
	var got sdktrace.ReadOnlySpan
	for _, s := range exp.spans {
		if strings.HasPrefix(s.Name(), "queue.process ") {
			got = s
		}
	}
	if got == nil {
		t.Fatal("delivery span missing")
	}
	if got.Name() != "queue.process score_candidate" {
		t.Fatalf("span name: %q", got.Name())
	}
	if got.Parent().IsValid() {
		t.Fatal("delivery span must be a root (link, not parent)")
	}
	if len(got.Links()) != 1 || got.Links()[0].SpanContext.SpanID() != pSpan.SpanContext().SpanID() {
		t.Fatalf("producer link missing or wrong: %+v", got.Links())
	}
	// The delivery span must be the active one in the handler ctx. Note
	// IsRecording() is false by inspection time — End() already ran.
	if handlerSpan == nil || handlerSpan.SpanContext().SpanID() != got.SpanContext().SpanID() {
		t.Fatal("handler did not run inside the delivery span")
	}
}

// Malformed/garbage traceparent must never break processing.
func TestTracingMiddlewareToleratesGarbageHeader(t *testing.T) {
	exp := &memExporter{}
	tp := newTestProvider(exp)
	prev := otel.GetTracerProvider()
	otel.SetTracerProvider(tp)
	t.Cleanup(func() { otel.SetTracerProvider(prev) })

	task := asynq.NewTaskWithHeaders("x", []byte("{}"), map[string]string{"traceparent": "garbage"})
	mw := TracingMiddleware()
	err := mw(asynq.HandlerFunc(func(context.Context, *asynq.Task) error { return nil })).ProcessTask(context.Background(), task)
	if err != nil {
		t.Fatalf("garbage header broke processing: %v", err)
	}
	if exp.count() != 1 {
		t.Fatalf("spans: %d want 1", exp.count())
	}
	if len(exp.spans[0].Links()) != 0 {
		t.Fatal("no link expected for garbage traceparent")
	}
}

// Errors mark the delivery span and propagate unchanged.
func TestTracingMiddlewareRecordsHandlerError(t *testing.T) {
	exp := &memExporter{}
	tp := newTestProvider(exp)
	prev := otel.GetTracerProvider()
	otel.SetTracerProvider(tp)
	t.Cleanup(func() { otel.SetTracerProvider(prev) })

	wantErr := context.DeadlineExceeded
	task := asynq.NewTask("failing", []byte("{}"))
	err := TracingMiddleware()(asynq.HandlerFunc(func(context.Context, *asynq.Task) error {
		return wantErr
	})).ProcessTask(context.Background(), task)
	if err != wantErr {
		t.Fatalf("error mutated: %v", err)
	}
	if exp.count() != 1 {
		t.Fatalf("spans: %d want 1", exp.count())
	}
	if code := exp.spans[0].Status().Code; codes.Error != code {
		t.Fatalf("span status code: %v want Error", code)
	}
}

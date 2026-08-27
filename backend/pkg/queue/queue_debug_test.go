package queue

import (
	"context"
	"testing"

	"github.com/hibiken/asynq"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Debug handler is a normal asynq delivery: one root span named after the
// task, status OK on success, and the handler runs inside it.
func TestDebugHandlerSpan(t *testing.T) {
	exp := &memExporter{}
	tp := newTestProvider(exp)
	prev := otel.GetTracerProvider()
	otel.SetTracerProvider(tp)
	t.Cleanup(func() { otel.SetTracerProvider(prev) })

	task := asynq.NewTaskWithHeaders("dbg", []byte("{}"), map[string]string{})

	var handlerSpan trace.Span
	err := TracingMiddleware()(asynq.HandlerFunc(func(ctx context.Context, t *asynq.Task) error {
		handlerSpan = trace.SpanFromContext(ctx)
		return nil
	})).ProcessTask(context.Background(), task)
	if err != nil {
		t.Fatalf("debug handler: %v", err)
	}

	if got := exp.count(); got != 1 {
		t.Fatalf("exported spans: %d want 1", got)
	}
	got := exp.spans[0]
	if want := "queue.process dbg"; got.Name() != want {
		t.Fatalf("span name: %q want %q", got.Name(), want)
	}
	if got.Status().Code == codes.Error {
		t.Fatalf("span status: %v want non-Error (success)", got.Status().Code)
	}
	if handlerSpan == nil || handlerSpan.SpanContext().SpanID() != got.SpanContext().SpanID() {
		t.Fatal("handler did not run inside the delivery span")
	}
	if got.Parent().IsValid() {
		t.Fatal("debug delivery span must be a root (link, not parent)")
	}
	if len(got.Links()) != 0 {
		t.Fatal("no producer link expected for debug task without traceparent")
	}
}

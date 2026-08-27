package queue

import (
	"context"

	"github.com/hibiken/asynq"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

var propagator = propagation.NewCompositeTextMapPropagator(propagation.TraceContext{})

// taskHeaderCarrier adapts asynq task headers to the propagation interface.
type taskHeaderCarrier map[string]string

func (c taskHeaderCarrier) Get(key string) string { return c[key] }
func (c taskHeaderCarrier) Set(key, value string) { c[key] = value }
func (c taskHeaderCarrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	return keys
}

// injectTraceContext writes the active span context into the task header so
// the worker delivery can link back (plan batch D). Typed payloads untouched.
func injectTraceContext(ctx context.Context, task *asynq.Task) {
	headers := task.Headers()
	if headers == nil {
		return
	}
	propagator.Inject(ctx, taskHeaderCarrier(headers))
}

// TracingMiddleware wraps worker dispatch: each delivery becomes a root span
// `queue.process <task>` LINKED to the producer trace. Links over parents —
// retries redeliver stale headers and a link states honest causality.
// Wire it FIRST on the worker mux so panic recovery nests inside the span.
func TracingMiddleware() func(asynq.Handler) asynq.Handler {
	return func(next asynq.Handler) asynq.Handler {
		return asynq.HandlerFunc(func(ctx context.Context, task *asynq.Task) error {
			var links []trace.Link
			if sc := trace.SpanContextFromContext(
				propagator.Extract(context.Background(), taskHeaderCarrier(task.Headers()))); sc.IsValid() {
				links = append(links, trace.Link{SpanContext: sc})
			}
			// Resolve the global provider per call so tests swapping the
			// provider late are honored (package-var tracers snapshot).
			ctx, span := otel.Tracer("github.com/intivai/backend/pkg/queue").Start(ctx,
				"queue.process "+task.Type(),
				trace.WithLinks(links...),
				trace.WithAttributes(attribute.String("messaging.task.name", task.Type())))
			defer span.End()

			err := next.ProcessTask(ctx, task)
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
			}
			return err
		})
	}
}

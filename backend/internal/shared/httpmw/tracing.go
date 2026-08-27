package httpmw

import (
	"github.com/gofiber/fiber/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// TracingConfig configures the HTTP server-span middleware.
type TracingConfig struct {
	// Provider overrides the global OTel tracer provider. Nil = global
	// (set once by pkg/telemetry.Init). Tests inject an in-memory provider.
	Provider trace.TracerProvider
}

// tracer resolves the global provider per call (mirrors pkg/db: a package-
// level tracer freezes onto the first installed provider, which is once-only).
func tracer() trace.Tracer {
	return otel.Tracer("github.com/intivai/backend/internal/shared/httpmw")
}

// Tracing is the HTTP server-span middleware for the Fiber app. It is a
// hand-rolled replacement for otelfiber: the contrib middleware unconditionally
// exports url.query, url.full, user-agent and Basic-auth enduser attributes,
// which leak ?ticket= / review tokens into traces (J2). The Go SDK cannot
// remove attributes once added, so the middleware imposes the safe attribute
// set itself. Semantics preserved from the previous wrapper:
//   - span name = "<METHOD> <route template>" — bounded cardinality, never
//     raw paths with IDs baked in (plan §5);
//   - scrape noise (/metrics, /healthz) is never traced;
//   - NOTHING from the URL query string or Authorization header is exported.
func Tracing(cfg TracingConfig) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if c.Path() == "/metrics" || c.Path() == "/healthz" {
			return c.Next()
		}

		spanName := c.Method() + " /"
		tr := tracer()
		if cfg.Provider != nil {
			tr = cfg.Provider.Tracer("github.com/intivai/backend/internal/shared/httpmw")
		}
		// Seed from UserContext: spans created later inside the handler chain
		// (including the WS upgrade path, J4) nest under this server span via
		// the request context the middleware sets.
		ctx, span := tr.Start(c.UserContext(), spanName, trace.WithSpanKind(trace.SpanKindServer))
		defer span.End()

		span.SetAttributes(
			attribute.String("http.request.method", c.Method()),
			attribute.String("http.client_ip", c.IP()),
		)
		if ua := string(c.Request().Header.UserAgent()); ua != "" {
			span.SetAttributes(attribute.String("user_agent.original", ua))
		}

		c.SetUserContext(ctx)

		// Treat the handler as non-remote: keep the request ctx attached for
		// downstream layers (tenant-tx middleware, repos) so their spans
		// parent correctly.
		err := c.Next()

		// Fiber populates c.Route() only after routing (otelfiber does the
		// same late rename, fiber.go:124): the route TEMPLATE is bounded
		// cardinality — never the raw path with IDs baked in.
		span.SetName(c.Method() + " " + c.Route().Path)
		span.SetAttributes(attribute.String("http.route", c.Route().Path))
		span.SetAttributes(attribute.Int64("http.response.status_code", int64(c.Response().StatusCode())))
		if c.Response().StatusCode() >= fiber.StatusInternalServerError {
			span.SetStatus(codes.Error, "http 5xx")
		} else if c.Response().StatusCode() >= fiber.StatusBadRequest {
			span.SetStatus(codes.Error, "http client error")
		}
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "http handler error")
		}
		return err
	}
}

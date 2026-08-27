package httpmw

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

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

func newTestApp(t *testing.T, exp *memExporter, logBuf *bytes.Buffer) *fiber.App {
	t.Helper()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exp))
	logger := zerolog.New(logBuf)

	app := fiber.New()
	app.Use(Tracing(TracingConfig{Provider: tp}))
	app.Use(RequestID(logger))
	app.Use(Audit(logger))

	app.Get("/api/v1/cvs/:id", func(c *fiber.Ctx) error {
		l, _ := c.Locals("logger").(zerolog.Logger)
		l.Info().Msg("inside-handler")
		return c.SendStatus(fiber.StatusOK)
	})
	app.Get("/healthz", func(c *fiber.Ctx) error { return c.SendString("ok") })
	return app
}

// Server span must use the ROUTE TEMPLATE as name (bounded cardinality),
// never the raw path with IDs baked in (plan §5).
func TestTracingSpanUsesRouteTemplate(t *testing.T) {
	exp := &memExporter{}
	var logBuf bytes.Buffer
	app := newTestApp(t, exp, &logBuf)

	resp, err := app.Test(httptest.NewRequest("GET", "/api/v1/cvs/550e8400-e29b-41d4-a716-446655440000", nil))
	if err != nil || resp.StatusCode != 200 {
		t.Fatalf("request: resp=%v err=%v", resp, err)
	}
	if exp.count() != 1 {
		t.Fatalf("spans exported: %d want 1", exp.count())
	}
	span := exp.spans[0]
	if want := "GET /api/v1/cvs/:id"; span.Name() != want {
		t.Fatalf("span name: got %q want %q (raw path leaks cardinality)", span.Name(), want)
	}
	if span.SpanKind() != trace.SpanKindServer {
		t.Fatalf("span kind: got %v want server", span.SpanKind())
	}
	for _, attr := range span.Attributes() {
		if attr.Key == "http.response.status_code" && attr.Value.AsInt64() != 200 {
			t.Fatalf("status attr: %v", attr.Value)
		}
	}
}

// The scoped request logger must carry the span's trace_id so a log line
// jumps straight to its waterfall (plan D6).
func TestRequestIDLoggerCarriesTraceID(t *testing.T) {
	exp := &memExporter{}
	var logBuf bytes.Buffer
	app := newTestApp(t, exp, &logBuf)

	_, err := app.Test(httptest.NewRequest("GET", "/api/v1/cvs/some-id", nil))
	if err != nil {
		t.Fatal(err)
	}
	if exp.count() != 1 {
		t.Fatalf("spans: %d", exp.count())
	}
	wantTrace := exp.spans[0].SpanContext().TraceID().String()

	firstLine := strings.SplitN(strings.TrimSpace(logBuf.String()), "\n", 2)[0]
	var line map[string]any
	if err := json.Unmarshal([]byte(firstLine), &line); err != nil {
		t.Fatalf("log line not json: %q (%v)", firstLine, err)
	}
	got, _ := line["trace_id"].(string)
	if got != wantTrace {
		t.Fatalf("trace_id correlation: log=%q span=%s", got, wantTrace)
	}
	if _, ok := line["request_id"]; !ok {
		t.Fatal("request_id lost when adding trace_id")
	}
}

// Metrics/health endpoints are scrape noise — never traced.
func TestTracingSkipsMetricsAndHealthz(t *testing.T) {
	exp := &memExporter{}
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exp))
	logger := zerolog.New(&bytes.Buffer{})

	app := fiber.New()
	app.Use(Tracing(TracingConfig{Provider: tp}))
	app.Use(RequestID(logger))
	app.Get("/metrics", func(c *fiber.Ctx) error { return c.SendString("m") })
	app.Get("/healthz", func(c *fiber.Ctx) error { return c.SendString("ok") })

	for _, path := range []string{"/metrics", "/healthz"} {
		_, _ = app.Test(httptest.NewRequest("GET", path, nil))
	}
	if exp.count() != 0 {
		t.Fatalf("skipped routes produced %d spans, want 0", exp.count())
	}
}

// J2: URL query/full and Authorization values MUST never reach exported span
// attributes — ?ticket= / review tokens ride the WS upgrade and every API
// call. The hand-rolled middleware exports a bounded safe attribute set only.
func TestTracingNeverExportsSensitiveURLOrAuthAttributes(t *testing.T) {
	exp := &memExporter{}
	app := newTestApp(t, exp, &bytes.Buffer{})

	req := httptest.NewRequest("GET", "/api/v1/cvs/secret-id?ticket=CANDIDATE_SECRET_QUERY_ABC&token=TOPSECRET", nil)
	req.Header.Set("Authorization", "Bearer CANDIDATE_SECRET_AUTH_XYZ")
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != 200 {
		t.Fatalf("request: resp=%v err=%v", resp, err)
	}
	if exp.count() != 1 {
		t.Fatalf("spans exported: %d want 1", exp.count())
	}
	span := exp.spans[0]
	for _, attr := range span.Attributes() {
		if strings.Contains(strings.ToLower(string(attr.Key)), "query") ||
			strings.Contains(strings.ToLower(string(attr.Key)), "full") {
			t.Fatalf("sensitive URL attribute exported: key=%q value=%q", attr.Key, attr.Value.AsString())
		}
		if strings.Contains(attr.Value.AsString(), "CANDIDATE_SECRET_QUERY_ABC") ||
			strings.Contains(attr.Value.AsString(), "TOPSECRET") ||
			strings.Contains(attr.Value.AsString(), "CANDIDATE_SECRET_AUTH_XYZ") {
			t.Fatalf("secret value leaked into span attribute: key=%q value=%q", attr.Key, attr.Value.AsString())
		}
	}
}

package observability

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"
	fibersentry "github.com/gofiber/contrib/fibersentry"
	"github.com/gofiber/fiber/v2"
	"github.com/intivai/backend/pkg/config"
	"github.com/intivai/backend/pkg/db"
)

// recordingTransport captures events synchronously so tests can assert on
// them without network I/O.
type recordingTransport struct {
	events []*sentry.Event
}

func (t *recordingTransport) Configure(sentry.ClientOptions) {}

func (t *recordingTransport) SendEvent(event *sentry.Event) {
	t.events = append(t.events, event)
}

func (t *recordingTransport) Flush(time.Duration) bool { return true }

func (t *recordingTransport) FlushWithContext(context.Context) bool { return true }

func (t *recordingTransport) Close() {}

// stubHub builds a hub whose client emits into the given transport, keyed by
// a parseable offline DSN (no traffic leaves the process — transport is
// replaced).
func stubHub(transport *recordingTransport) *sentry.Hub {
	client, err := sentry.NewClient(sentry.ClientOptions{
		Dsn:       "http://public@example.com/1",
		Transport: transport,
	})
	if err != nil {
		panic("test DSN must parse: " + err.Error())
	}
	return sentry.NewHub(client, sentry.NewScope())
}

// resetGlobalHub restores a disabled global hub so tests that assume no
// Sentry client stay order-independent.
func resetGlobalHub(t *testing.T) {
	t.Cleanup(func() {
		if err := sentry.Init(sentry.ClientOptions{}); err != nil {
			t.Fatalf("reset global sentry hub: %v", err)
		}
	})
}

func TestInitEmptyDSNIsNoOp(t *testing.T) {
	cfg := &config.Config{}
	if err := Init(cfg); err != nil {
		t.Fatalf("empty DSN must disable Sentry without error, got %v", err)
	}
	if client := sentry.CurrentHub().Client(); client != nil {
		t.Fatal("empty DSN must leave the global hub without a client")
	}
	if got := CaptureError(context.Background(), errors.New("boom")); got != "" {
		t.Fatalf("CaptureError must be a no-op without a client, got event %s", got)
	}
	if got := CapturePanic(context.Background(), "boom"); got != "" {
		t.Fatalf("CapturePanic must be a no-op without a client, got event %s", got)
	}
}

func TestInitGarbageDSNReturnsError(t *testing.T) {
	cfg := &config.Config{}
	cfg.Sentry.DSN = "not-a-sentry-dsn"
	err := Init(cfg)
	if err == nil {
		t.Fatal("garbage DSN must surface an init error, got nil")
	}
	if client := sentry.CurrentHub().Client(); client != nil {
		t.Fatal("failed init must leave the global hub clientless")
	}
}

func TestTracesSampleRateEnvParsing(t *testing.T) {
	tests := []struct {
		name string
		env  string
		want float64
		err  bool
	}{
		{name: "unset defaults to modest rate", env: "", want: DefaultTracesSampleRate},
		{name: "valid override", env: "0.25", want: 0.25},
		{name: "zero allowed", env: "0", want: 0},
		{name: "one allowed", env: "1", want: 1},
		{name: "non-numeric rejected", env: "abc", err: true},
		{name: "above one rejected", env: "1.5", err: true},
		{name: "negative rejected", env: "-0.1", err: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(sampleRateEnv, tt.env)
			got, err := tracesSampleRate()
			if tt.err {
				if err == nil {
					t.Fatalf("%q must be rejected, got rate %v", tt.env, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("%q must parse, got %v", tt.env, err)
			}
			if got != tt.want {
				t.Fatalf("rate = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCaptureErrorAttachesTagsFromContext(t *testing.T) {
	transport := &recordingTransport{}
	hub := stubHub(transport)

	ctx := context.Background()
	ctx = context.WithValue(ctx, sentry.HubContextKey, hub)
	ctx = db.WithTenant(ctx, "org-123")
	ctx = context.WithValue(ctx, requestIDKey{}, "req-42")

	eventID := CaptureError(ctx, errors.New("boom"))
	if eventID == "" {
		t.Fatal("CaptureError must return an event ID when enabled")
	}
	if len(transport.events) != 1 {
		t.Fatalf("want exactly 1 captured event, got %d", len(transport.events))
	}
	event := transport.events[0]
	if event.Exception[0].Value != "boom" {
		t.Fatalf("exception value = %q, want boom", event.Exception[0].Value)
	}
	if event.Tags["org_id"] != "org-123" {
		t.Fatalf("org_id tag = %q, want org-123", event.Tags["org_id"])
	}
	if event.Tags["request_id"] != "req-42" {
		t.Fatalf("request_id tag = %q, want req-42", event.Tags["request_id"])
	}
}

func TestCaptureErrorHonorsPlainStringRequestID(t *testing.T) {
	transport := &recordingTransport{}
	hub := stubHub(transport)

	ctx := context.WithValue(context.Background(), sentry.HubContextKey, hub)
	//nolint:staticcheck // SA1029 intentional: exercising legacy plain-string key compat
	ctx = context.WithValue(ctx, requestIDStringKey, "req-plain")

	if CaptureError(ctx, errors.New("boom")) == "" {
		t.Fatal("CaptureError must capture when enabled")
	}
	if len(transport.events) != 1 || transport.events[0].Tags["request_id"] != "req-plain" {
		t.Fatalf("request_id tag from plain string key not attached: %+v", transport.events)
	}
}

func TestCaptureErrorNilErrorIsNoOp(t *testing.T) {
	transport := &recordingTransport{}
	hub := stubHub(transport)
	ctx := context.WithValue(context.Background(), sentry.HubContextKey, hub)

	if got := CaptureError(ctx, nil); got != "" {
		t.Fatalf("nil error must not capture, got event %s", got)
	}
	if len(transport.events) != 0 {
		t.Fatalf("nil error must not produce events, got %d", len(transport.events))
	}
}

func TestCapturePanicRecordsRecoveredValue(t *testing.T) {
	transport := &recordingTransport{}
	hub := stubHub(transport)
	ctx := context.WithValue(context.Background(), sentry.HubContextKey, hub)

	if CapturePanic(ctx, "worker exploded") == "" {
		t.Fatal("CapturePanic must return an event ID when enabled")
	}
	if len(transport.events) != 1 {
		t.Fatalf("want exactly 1 captured panic event, got %d", len(transport.events))
	}
	// String panic values land in Message; error values land in Exception.
	event := transport.events[0]
	if event.Message != "worker exploded" &&
		(len(event.Exception) == 0 || event.Exception[0].Value != "worker exploded") {
		t.Fatalf("panic value missing from event: message=%q exceptions=%v",
			event.Message, event.Exception)
	}
}

func TestFiberContextCarriesHubAndRequestID(t *testing.T) {
	resetGlobalHub(t)
	transport := &recordingTransport{}
	var seen []*sentry.Event
	if err := sentry.Init(sentry.ClientOptions{
		Dsn:              "http://public@example.com/1",
		AttachStacktrace: true,
		BeforeSend: func(event *sentry.Event, _ *sentry.EventHint) *sentry.Event {
			seen = append(seen, event)
			return event
		},
		Transport: transport,
	}); err != nil {
		t.Fatalf("global init with test transport: %v", err)
	}

	app := fiber.New()
	app.Use(fibersentry.New())
	app.Get("/fail", func(c *fiber.Ctx) error {
		c.Locals("request_id", "req-fiber")
		CaptureError(FiberContext(c), errors.New("handler blew up"))
		return nil
	})

	httpReq := httptest.NewRequest(fiber.MethodGet, "/fail", nil)
	resp, err := app.Test(httpReq)
	if err != nil {
		t.Fatalf("fiber test request: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if len(seen) != 1 {
		t.Fatalf("want exactly 1 event via BeforeSend, got %d", len(seen))
	}
	if seen[0].Tags["request_id"] != "req-fiber" {
		t.Fatalf("request_id tag = %q, want req-fiber", seen[0].Tags["request_id"])
	}
	if seen[0].Exception[0].Value != "handler blew up" {
		t.Fatalf("exception value = %q", seen[0].Exception[0].Value)
	}
}

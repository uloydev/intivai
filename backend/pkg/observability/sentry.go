// Package observability wires Sentry error reporting for the Go backend:
// process-wide init from config, Fiber middleware glue, and capture helpers
// that attach request/tenant correlation tags when they are present in the
// context. Everything degrades to a no-op when no DSN is configured.
package observability

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/getsentry/sentry-go"
	fibersentry "github.com/gofiber/contrib/fibersentry"
	"github.com/gofiber/fiber/v2"
	"github.com/intivai/backend/pkg/config"
	"github.com/intivai/backend/pkg/db"
)

// DefaultTracesSampleRate keeps performance tracing at a modest volume;
// override with INTIVAI_SENTRY_TRACES_SAMPLE_RATE (number in [0, 1]).
const DefaultTracesSampleRate = 0.1

const sampleRateEnv = "INTIVAI_SENTRY_TRACES_SAMPLE_RATE"

// requestIDKey mirrors the value httpmw.RequestID stores in fiber locals;
// FiberContext promotes it into the capture context. Plain string-keyed
// "request_id" context values are honored too, so non-Fiber callers
// (workers, services) can attach the tag without importing fiber.
type requestIDKey struct{}

const requestIDStringKey = "request_id"

type settings struct {
	release        string
	mutateClientOp func(*sentry.ClientOptions)
}

// Option adjusts Init behavior.
type Option func(*settings)

// WithRelease sets the release identifier (typically injected via
// -ldflags "-X main.version=..." into cmd/server).
func WithRelease(release string) Option {
	return func(s *settings) { s.release = release }
}

// WithClientOptions lets tests inject a stub Transport (or otherwise tweak
// the raw sentry.ClientOptions) before the client is created.
func WithClientOptions(mutate func(*sentry.ClientOptions)) Option {
	return func(s *settings) { s.mutateClientOp = mutate }
}

// Init initializes the global Sentry hub from config. A missing DSN disables
// reporting entirely (nil error, helpers become no-ops). An invalid DSN or
// sample rate returns an error — callers should warn and continue rather
// than crash startup over telemetry.
func Init(cfg *config.Config, opts ...Option) error {
	if cfg.Sentry.DSN == "" {
		return nil
	}
	s := settings{release: "dev"}
	for _, opt := range opts {
		opt(&s)
	}
	rate, err := tracesSampleRate()
	if err != nil {
		return err
	}
	clientOpts := sentry.ClientOptions{
		Dsn:              cfg.Sentry.DSN,
		Environment:      cfg.App.Env,
		Release:          s.release,
		TracesSampleRate: rate,
		AttachStacktrace: true,
	}
	if s.mutateClientOp != nil {
		s.mutateClientOp(&clientOpts)
	}
	return sentry.Init(clientOpts)
}

// Flush waits up to timeout for queued events to be delivered; call on
// graceful shutdown.
func Flush(timeout time.Duration) bool {
	return sentry.Flush(timeout)
}

func tracesSampleRate() (float64, error) {
	raw := os.Getenv(sampleRateEnv)
	if raw == "" {
		return DefaultTracesSampleRate, nil
	}
	rate, err := strconv.ParseFloat(raw, 64)
	if err != nil || rate < 0 || rate > 1 {
		return 0, fmt.Errorf("invalid %s %q: must be a number in [0, 1]", sampleRateEnv, raw)
	}
	return rate, nil
}

// CaptureError reports err to Sentry through the request-scoped hub when the
// context carries one (see FiberContext), falling back to the global hub.
// The org_id tag is taken from the tenant context (db.WithTenant /
// tenant-tx middleware) and request_id from the promoted fiber local or a
// plain "request_id" value. Returns the event ID, or "" when Sentry is
// disabled or err is nil.
func CaptureError(ctx context.Context, err error) string {
	if err == nil {
		return ""
	}
	hub := resolveHub(ctx)
	if hub.Client() == nil {
		return ""
	}
	var eventID *sentry.EventID
	hub.WithScope(func(scope *sentry.Scope) {
		applyCorrelationTags(ctx, scope)
		eventID = hub.CaptureException(err)
	})
	if eventID == nil {
		return ""
	}
	return string(*eventID)
}

// CapturePanic reports a recovered panic value to Sentry (with the goroutine
// stack thanks to AttachStacktrace) and returns the event ID, or "" when
// Sentry is disabled. Callers decide what happens next (log + convert to
// error, re-panic, ...).
func CapturePanic(ctx context.Context, r any) string {
	// r is deliberately `any`: recover() yields an arbitrary panic value.
	hub := resolveHub(ctx)
	if hub.Client() == nil {
		return ""
	}
	eventID := hub.RecoverWithContext(ctx, r)
	if eventID == nil {
		return ""
	}
	return string(*eventID)
}

func resolveHub(ctx context.Context) *sentry.Hub {
	if hub := sentry.GetHubFromContext(ctx); hub != nil {
		return hub
	}
	return sentry.CurrentHub()
}

func applyCorrelationTags(ctx context.Context, scope *sentry.Scope) {
	if orgID, ok := db.TenantFrom(ctx); ok {
		scope.SetTag("org_id", orgID)
	}
	if rid := requestIDFrom(ctx); rid != "" {
		scope.SetTag("request_id", rid)
	}
}

func requestIDFrom(ctx context.Context) string {
	if rid, ok := ctx.Value(requestIDKey{}).(string); ok && rid != "" {
		return rid
	}
	if rid, ok := ctx.Value(requestIDStringKey).(string); ok && rid != "" {
		return rid
	}
	return ""
}

// FiberContext promotes the values fibersentry and httpmw.RequestID park in
// fiber locals (per-request Sentry hub, X-Request-Id) into the request
// context, so CaptureError can resolve them downstream. Pass its result as
// the ctx argument from fiber handlers (e.g. the central error handler).
func FiberContext(c *fiber.Ctx) context.Context {
	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}
	if hub := fibersentry.GetHubFromContext(c); hub != nil {
		ctx = context.WithValue(ctx, sentry.HubContextKey, hub)
	}
	if rid, ok := c.Locals("request_id").(string); ok && rid != "" {
		ctx = context.WithValue(ctx, requestIDKey{}, rid)
	}
	return ctx
}

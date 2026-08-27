package llm

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Tracer resolves the global provider PER CALL — package-level otel.Tracer
// vars freeze onto the first provider ever installed (global delegation is
// once-only), breaking tests and any late Init.
func tracer() trace.Tracer {
	return otel.Tracer("github.com/intivai/backend/internal/llm")
}

// recordErr marks a span errored with a SANITIZED class — provider bodies can
// contain PII/customer content (J9), so err.Error() must never enter span
// status descriptions, attributes, or exception events.
func recordErr(span trace.Span, err error) {
	if err == nil {
		return
	}
	class := safeErrorClass(err)
	span.AddEvent("llm.error", trace.WithAttributes(
		attribute.String("llm.error.class", class),
		attribute.Bool("llm.error.retryable", isRetryable(err)),
	))
	span.SetStatus(codes.Error, class)
}

// safeErrorClass maps a provider error to a coarse class with no payload
// content. Exact class names are the contract for dashboards/alerts.
func safeErrorClass(err error) string {
	if err == nil {
		return "unknown"
	}
	switch {
	case errors.Is(err, ErrRateLimited):
		return "rate_limited"
	case errors.Is(err, ErrUpstream):
		return "upstream"
	case errors.Is(err, ErrStructuredParse):
		return "bad_request"
	case errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
		return "timeout"
	}
	var ne net.Error
	if errors.As(err, &ne) {
		if ne.Timeout() {
			return "timeout"
		}
		return "network"
	}
	return "unknown"
}

func requestAttrs(req ChatRequest) []trace.SpanStartOption {
	attrs := []attribute.KeyValue{attribute.String("llm.model", req.Model)}
	if req.OrgID != "" {
		attrs = append(attrs, attribute.String("org.id", req.OrgID))
	}
	return []trace.SpanStartOption{trace.WithAttributes(attrs...)}
}

// Client wraps a primary provider with retry + exponential backoff + fallback.
type Client struct {
	primary    Provider
	fallback   Provider
	ledger     TokenLedger
	maxRetries int
	onRetry    func(attempt int, err error)
}

func NewClient(primary, fallback Provider, ledger TokenLedger, maxRetries int) *Client {
	if maxRetries <= 0 {
		maxRetries = 3
	}
	return &Client{primary: primary, fallback: fallback, ledger: ledger, maxRetries: maxRetries}
}

// OnRetry registers a hook (metrics/logging).
func (c *Client) OnRetry(fn func(attempt int, err error)) { c.onRetry = fn }

// Budget pre-charges: conservative token estimates reserved on the ledger
// before a call and trued up against real usage afterwards (Chat only).
const (
	chatBudgetEstimate          = 500
	chatStreamBudgetEstimate    = 1500
	structuredOutputBudgetGuess = 800
)

func (c *Client) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	// Op span + per-attempt children: retry storms and fallback handoffs
	// visible in one waterfall. Payload content never enters attributes.
	ctx, span := tracer().Start(ctx, "llm.chat", requestAttrs(req)...)
	defer span.End()

	if c.ledger != nil && req.OrgID != "" {
		// Pre-flight check with a fixed estimate (e.g., 500) just to fail fast if totally out of budget.
		if err := c.ledger.CheckAndRecord(ctx, req.OrgID, chatBudgetEstimate); err != nil {
			recordErr(span, err)
			return nil, err
		}
	}

	var resp *ChatResponse
	var err error
	for attempt := 0; attempt < c.maxRetries; attempt++ {
		actx, attemptSpan := tracer().Start(ctx, fmt.Sprintf("llm.attempt %d", attempt+1),
			trace.WithAttributes(attribute.Int("llm.attempt", attempt+1)))
		resp, err = c.primary.Chat(actx, req)
		recordErr(attemptSpan, err)
		attemptSpan.End()
		if err == nil {
			break
		}
		if c.onRetry != nil {
			c.onRetry(attempt+1, err)
		}
		if !isRetryable(err) {
			break
		}
		if attempt < c.maxRetries-1 {
			backoff := time.Duration(math.Pow(2, float64(attempt))) * time.Second
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				recordErr(span, ctx.Err())
				return nil, ctx.Err()
			}
		}
	}

	// Fallback only for retryable failures — a non-retryable primary error
	// (e.g. invalid request) must not be masked by a fallback attempt.
	if err != nil && c.fallback != nil && isRetryable(err) {
		fctx, fspan := tracer().Start(ctx, "llm.fallback")
		resp, err = c.fallback.Chat(fctx, req)
		recordErr(fspan, err)
		fspan.End()
	}

	if err != nil {
		recordErr(span, err)
		return nil, fmt.Errorf("all providers failed after %d attempts: %w", c.maxRetries, err)
	}

	if c.ledger != nil && req.OrgID != "" {
		total := resp.Usage.PromptTokens + resp.Usage.CompletionTokens
		diff := total - chatBudgetEstimate
		if diff > 0 {
			// Best-effort true-up: use the original ctx but ignore errors below timeout so we never double-charge on retry.
			trueCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if ctxErr := ctx.Err(); ctxErr == nil {
				_ = c.ledger.CheckAndRecord(trueCtx, req.OrgID, diff)
			}
		}
	}

	return resp, nil
}

func (c *Client) ChatStream(ctx context.Context, req ChatRequest) (<-chan string, error) {
	// Setup span: ends when the channel is handed over — consumption latency
	// belongs to the WS turn spans (plan batch E).
	ctx, span := tracer().Start(ctx, "llm.chat_stream", requestAttrs(req)...)
	defer span.End()

	if c.ledger != nil && req.OrgID != "" {
		if err := c.ledger.CheckAndRecord(ctx, req.OrgID, chatStreamBudgetEstimate); err != nil {
			recordErr(span, err)
			return nil, err
		}
	}

	var lastErr error
	for attempt := 0; attempt < c.maxRetries; attempt++ {
		actx, attemptSpan := tracer().Start(ctx, fmt.Sprintf("llm.attempt %d", attempt+1),
			trace.WithAttributes(attribute.Int("llm.attempt", attempt+1)))
		ch, err := c.primary.ChatStream(actx, req)
		recordErr(attemptSpan, err)
		attemptSpan.End()
		if err == nil {
			return ch, nil
		}
		lastErr = err
		if c.onRetry != nil {
			c.onRetry(attempt+1, err)
		}
		if !isRetryable(err) {
			break
		}
		if attempt < c.maxRetries-1 {
			backoff := time.Duration(math.Pow(2, float64(attempt))) * time.Second
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				recordErr(span, ctx.Err())
				return nil, ctx.Err()
			}
		}
	}

	// Fallback only for retryable failures — a non-retryable primary error
	// (e.g. invalid request) must not be masked by a fallback attempt.
	if c.fallback != nil && lastErr != nil && isRetryable(lastErr) {
		fctx, fspan := tracer().Start(ctx, "llm.fallback")
		for attempt := 0; attempt < c.maxRetries; attempt++ {
			actx, attemptSpan := tracer().Start(fctx, fmt.Sprintf("llm.attempt %d", attempt+1),
				trace.WithAttributes(attribute.Int("llm.attempt", attempt+1)))
			ch, err := c.fallback.ChatStream(actx, req)
			recordErr(attemptSpan, err)
			attemptSpan.End()
			if err == nil {
				fspan.End()
				return ch, nil
			}
			lastErr = err
			if c.onRetry != nil {
				c.onRetry(attempt+1, err)
			}
			if !isRetryable(err) {
				break
			}
			if attempt < c.maxRetries-1 {
				backoff := time.Duration(math.Pow(2, float64(attempt))) * time.Second
				select {
				case <-time.After(backoff):
				case <-fctx.Done():
					recordErr(fspan, fctx.Err())
					fspan.End()
					return nil, fctx.Err()
				}
			}
		}
		fspan.End()
	}

	if lastErr != nil {
		recordErr(span, lastErr)
		return nil, fmt.Errorf("all providers failed after %d attempts: %w", c.maxRetries, lastErr)
	}
	recordErr(span, ErrUpstream)
	return nil, fmt.Errorf("all providers failed after %d attempts: %w", c.maxRetries, ErrUpstream)
}

func (c *Client) StructuredOutput(ctx context.Context, req StructuredRequest) (any, error) {
	ctx, span := tracer().Start(ctx, "llm.structured_output", trace.WithAttributes(
		attribute.String("llm.model", req.Model),
	))
	defer span.End()

	if c.ledger != nil && req.OrgID != "" {
		if err := c.ledger.CheckAndRecord(ctx, req.OrgID, structuredOutputBudgetGuess); err != nil {
			recordErr(span, err)
			return nil, err
		}
	}

	out, err := c.primary.StructuredOutput(ctx, req)
	if err != nil && c.fallback != nil {
		out, err = c.fallback.StructuredOutput(ctx, req)
		recordErr(span, err)
		return out, err
	}
	recordErr(span, err)
	return out, err
}

func (c *Client) Embed(ctx context.Context, text string) ([]float32, error) {
	return c.primary.Embed(ctx, text)
}

func (c *Client) CountTokens(text string) int { return c.primary.CountTokens(text) }

func isRetryable(err error) bool {
	return errors.Is(err, ErrRateLimited) || errors.Is(err, ErrUpstream)
}

var (
	ErrRateLimited = errors.New("llm rate limited")
	ErrUpstream    = errors.New("llm upstream error")
	// ErrStructuredParse — the provider responded but the payload was not
	// valid JSON for the requested schema. Permanent: retrying cannot fix it.
	ErrStructuredParse = errors.New("llm structured output parse failed")
)

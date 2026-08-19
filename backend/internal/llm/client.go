package llm

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"
)

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

func (c *Client) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	if c.ledger != nil && req.OrgID != "" {
		// Pre-flight check with a fixed estimate (e.g., 500) just to fail fast if totally out of budget.
		if err := c.ledger.CheckAndRecord(ctx, req.OrgID, 500); err != nil {
			return nil, err
		}
	}

	var resp *ChatResponse
	var err error
	for attempt := 0; attempt < c.maxRetries; attempt++ {
		resp, err = c.primary.Chat(ctx, req)
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
				return nil, ctx.Err()
			}
		}
	}

	if err != nil && c.fallback != nil {
		resp, err = c.fallback.Chat(ctx, req)
	}

	if err != nil {
		return nil, fmt.Errorf("all providers failed after %d attempts: %w", c.maxRetries, err)
	}

	if c.ledger != nil && req.OrgID != "" {
		// Post-flight true-up: we pre-charged 500, so we record the difference
		total := resp.Usage.PromptTokens + resp.Usage.CompletionTokens
		diff := total - 500
		if diff > 0 {
			_ = c.ledger.CheckAndRecord(context.Background(), req.OrgID, diff)
		}
	}

	return resp, nil
}

func (c *Client) ChatStream(ctx context.Context, req ChatRequest) (<-chan string, error) {
	if c.ledger != nil && req.OrgID != "" {
		if err := c.ledger.CheckAndRecord(ctx, req.OrgID, 1500); err != nil {
			return nil, err
		}
	}

	ch, err := c.primary.ChatStream(ctx, req)
	if err == nil {
		return ch, nil
	}
	if isRetryable(err) && c.fallback != nil {
		return c.fallback.ChatStream(ctx, req)
	}
	return nil, err
}

func (c *Client) StructuredOutput(ctx context.Context, req StructuredRequest) (any, error) {
	if c.ledger != nil && req.OrgID != "" {
		if err := c.ledger.CheckAndRecord(ctx, req.OrgID, 800); err != nil {
			return nil, err
		}
	}

	out, err := c.primary.StructuredOutput(ctx, req)
	if err != nil && c.fallback != nil {
		return c.fallback.StructuredOutput(ctx, req)
	}
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

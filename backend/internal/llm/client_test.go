package llm

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"
)

type flakyProvider struct {
	mockProvider
	fails int
	calls int
}

func (f *flakyProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	f.calls++
	if f.calls <= f.fails {
		return nil, ErrRateLimited
	}
	return &ChatResponse{Content: "ok"}, nil
}

func TestClientChatRetriesTransientFailures(t *testing.T) {
	primary := &flakyProvider{fails: 2}
	client := NewClient(primary, nil, 3)
	resp, err := client.Chat(context.Background(), ChatRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Content != "ok" || primary.calls != 3 {
		t.Fatalf("calls=%d resp=%q", primary.calls, resp.Content)
	}
}

func TestClientChatFallsBackAfterRetries(t *testing.T) {
	primary := &flakyProvider{fails: 99}
	fallback := &mockProvider{chunks: []string{"fb"}}
	client := NewClient(primary, fallback, 2)
	resp, err := client.Chat(context.Background(), ChatRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Content != "fb" {
		t.Fatalf("resp = %q", resp.Content)
	}
}

func TestClientChatDoesNotRetryNonTransient(t *testing.T) {
	count := 0
	primary := &countingProvider{fn: func(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
		count++
		return nil, errors.New("bad request")
	}}
	client := NewClient(primary, nil, 5)
	_, err := client.Chat(context.Background(), ChatRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
	if count > 1 {
		t.Fatalf("non-transient error retried %d times", count)
	}
}

type countingProvider struct {
	fn func(ctx context.Context, req ChatRequest) (*ChatResponse, error)
}

func (c *countingProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	return c.fn(ctx, req)
}
func (c *countingProvider) ChatStream(ctx context.Context, req ChatRequest) (<-chan string, error) {
	return nil, errors.New("unused")
}
func (c *countingProvider) StructuredOutput(ctx context.Context, req StructuredRequest) (any, error) {
	return nil, errors.New("unused")
}
func (c *countingProvider) Embed(ctx context.Context, text string) ([]float32, error) {
	return nil, errors.New("unused")
}
func (c *countingProvider) CountTokens(text string) int { return 0 }

func TestClientChatAllProvidersFailed(t *testing.T) {
	primary := &flakyProvider{fails: 99}
	client := NewClient(primary, nil, 2)
	_, err := client.Chat(context.Background(), ChatRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestStatusErrorMapping(t *testing.T) {
	resp := &http.Response{StatusCode: http.StatusTooManyRequests, Body: nopCloser("slow down")}
	if !errors.Is(statusError(resp), ErrRateLimited) {
		t.Fatal("429 not mapped to ErrRateLimited")
	}
	resp = &http.Response{StatusCode: http.StatusBadGateway, Body: nopCloser("boom")}
	if !errors.Is(statusError(resp), ErrUpstream) {
		t.Fatal("502 not mapped to ErrUpstream")
	}
	resp = &http.Response{StatusCode: http.StatusUnauthorized, Body: nopCloser("no key")}
	if errors.Is(statusError(resp), ErrRateLimited) || errors.Is(statusError(resp), ErrUpstream) {
		t.Fatal("401 mapped as transient")
	}
}

func TestCountTokensFallback(t *testing.T) {
	p := &DeepSeekProvider{}
	if n := p.CountTokens(""); n != 0 {
		t.Fatalf("empty tokens = %d", n)
	}
	if n := p.CountTokens("hello world"); n == 0 {
		t.Fatal("fallback token count = 0")
	}
}

func TestStructuredOutputDelegates(t *testing.T) {
	mock := &mockProvider{structured: "delegated"}
	client := NewClient(mock, nil, 1)
	res, err := client.StructuredOutput(context.Background(), StructuredRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if res != "delegated" {
		t.Fatalf("structured = %v", res)
	}
}

func nopCloser(s string) *nopReadCloser {
	return &nopReadCloser{data: []byte(s)}
}

type nopReadCloser struct{ data []byte }

func (n *nopReadCloser) Read(p []byte) (int, error) {
	if len(n.data) == 0 {
		return 0, fmt.Errorf("EOF")
	}
	n2 := copy(p, n.data)
	n.data = n.data[n2:]
	return n2, nil
}
func (n *nopReadCloser) Close() error { return nil }

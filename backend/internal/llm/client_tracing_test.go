package llm

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func otelSetTracer(tp *sdktrace.TracerProvider) func() {
	prev := otel.GetTracerProvider()
	otel.SetTracerProvider(tp)
	return func() { otel.SetTracerProvider(prev) }
}

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

// flakyProvider fails n times with a retryable error, then succeeds.
type flakySpanProvider struct {
	failures int
	calls    int
}

func (f *flakySpanProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	f.calls++
	if f.calls <= f.failures {
		return nil, ErrUpstream // retryable
	}
	return &ChatResponse{Content: "ok"}, nil
}

func (*flakySpanProvider) ChatStream(context.Context, ChatRequest) (<-chan string, error) {
	ch := make(chan string, 1)
	ch <- "x"
	return ch, nil
}

func (*flakySpanProvider) StructuredOutput(context.Context, StructuredRequest) (any, error) {
	return nil, nil
}

func (*flakySpanProvider) Embed(context.Context, string) ([]float32, error) { return nil, nil }
func (*flakySpanProvider) CountTokens(string) int                           { return 0 }

// One op span per Chat call, one child span per provider attempt — retry
// visibility without prompt/completion content anywhere in attributes.
func TestChatEmitsOpAndAttemptSpans(t *testing.T) {
	exp := &memExporter{}
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exp))
	prev := otelSetTracer(tp)
	t.Cleanup(func() { prev() })

	primary := &flakySpanProvider{failures: 2}
	client := NewClient(primary, nil, nil, 3)
	_, err := client.Chat(context.Background(), ChatRequest{OrgID: "org-1", Model: "test-model", Messages: []Message{{Role: "user", Content: "secret prompt"}}})
	if err != nil {
		t.Fatal(err)
	}
	if primary.calls != 3 {
		t.Fatalf("provider calls: %d want 3", primary.calls)
	}

	var op *sdktrace.ReadOnlySpan
	attemptCount := 0
	for i := range exp.spans {
		s := exp.spans[i]
		switch {
		case s.Name() == "llm.chat":
			op = &exp.spans[i]
		case strings.HasPrefix(s.Name(), "llm.attempt"):
			if s.Status().Code == codes.Error && !strings.Contains(attrString(s, "error.message"), "secret") {
				attemptCount++
			} else if s.Status().Code != codes.Error {
				attemptCount++
			}
		}
	}
	if op == nil {
		t.Fatal("llm.chat op span missing")
	}
	if got := attrString(*op, "llm.model"); got != "test-model" {
		t.Fatalf("llm.model attr: %q want test-model", got)
	}
	if attemptCount != 3 {
		names := make([]string, 0, len(exp.spans))
		for _, s := range exp.spans {
			names = append(names, s.Name())
		}
		t.Fatalf("attempt spans: %d want 3; exported %v", attemptCount, names)
	}
	for _, s := range exp.spans {
		for _, kv := range s.Attributes() {
			if strings.Contains(kv.Value.AsString(), "secret") || strings.Contains(kv.Value.AsString(), "ok") {
				t.Fatalf("payload content leaked into span attr %v=%q", kv.Key, kv.Value.AsString())
			}
		}
	}
}

func attrString(s sdktrace.ReadOnlySpan, key string) string {
	for _, kv := range s.Attributes() {
		if string(kv.Key) == key {
			return kv.Value.AsString()
		}
	}
	return ""
}

// J9: the raw provider error body must never reach span status descriptions,
// events, or attributes. The error message here contains a synthetic secret
// the way real provider bodies do (e.g. an upstream HTML error page).
func TestErrorSpanNeverLeaksProviderBody(t *testing.T) {
	const secret = "CANDIDATE_SECRET_PAYLOAD123"
	exp := &memExporter{}
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exp))
	prev := otelSetTracer(tp)
	t.Cleanup(func() { prev() })

	leaky := leakyRateLimitedProvider{}
	client := NewClient(leaky, nil, nil, 1)
	_, err := client.Chat(context.Background(), ChatRequest{Model: "m"})
	if err == nil {
		t.Fatal("expected failure")
	}
	if !errors.Is(err, ErrRateLimited) {
		t.Fatalf("wrapped err: %v", err)
	}

	if len(exp.spans) == 0 {
		t.Fatal("no spans exported")
	}
	for _, s := range exp.spans {
		// Status: description may not carry the secret.
		if strings.Contains(s.Status().Description, secret) {
			t.Fatalf("provider body leaked into span status: %q", s.Status().Description)
		}
		// Exception events (RecordError) are gone — replaced by llm.error.
		for _, e := range s.Events() {
			if strings.Contains(e.Name, "exception") {
				t.Fatalf("RecordError exception event still present: %s", e.Name)
			}
			for _, kv := range e.Attributes {
				if strings.Contains(kv.Value.AsString(), secret) {
					t.Fatalf("provider body leaked into event %s attr %s", e.Name, kv.Key)
				}
			}
		}
		// Attributes may not carry it either.
		for _, kv := range s.Attributes() {
			if strings.Contains(kv.Value.AsString(), secret) {
				t.Fatalf("provider body leaked into span attr %v=%q", kv.Key, kv.Value.AsString())
			}
		}
	}

	// The op span carries the sanitized error class.
	found := false
	for _, s := range exp.spans {
		if s.Name() != "llm.chat" {
			continue
		}
		found = true
		if s.Status().Code != codes.Error {
			t.Fatalf("op span status code: %v want Error", s.Status().Code)
		}
		if got := s.Status().Description; got != "rate_limited" {
			t.Fatalf("op span status description: %q want rate_limited", got)
		}
		class, retry, eventFound := "", "", false
		for _, e := range s.Events() {
			if e.Name != "llm.error" {
				continue
			}
			eventFound = true
			for _, kv := range e.Attributes {
				switch string(kv.Key) {
				case "llm.error.class":
					class = kv.Value.AsString()
				case "llm.error.retryable":
					if kv.Value.AsBool() {
						retry = "true"
					}
				}
			}
		}
		if !eventFound {
			t.Fatal("llm.error event missing on op span")
		}
		if class != "rate_limited" {
			t.Fatalf("llm.error.class: %q want rate_limited", class)
		}
		if retry != "true" {
			t.Fatalf("llm.error.retryable: %q want true", retry)
		}
	}
	if !found {
		t.Fatal("llm.chat op span missing")
	}
}

type leakyRateLimitedProvider struct{}

func (leakyRateLimitedProvider) Chat(context.Context, ChatRequest) (*ChatResponse, error) {
	return nil, fmt.Errorf("%w: body CANDIDATE_SECRET_PAYLOAD123", ErrRateLimited)
}

func (leakyRateLimitedProvider) ChatStream(context.Context, ChatRequest) (<-chan string, error) {
	return nil, fmt.Errorf("%w: body CANDIDATE_SECRET_PAYLOAD123", ErrRateLimited)
}

func (leakyRateLimitedProvider) StructuredOutput(context.Context, StructuredRequest) (any, error) {
	return nil, fmt.Errorf("%w: body CANDIDATE_SECRET_PAYLOAD123", ErrRateLimited)
}

func (leakyRateLimitedProvider) Embed(context.Context, string) ([]float32, error) {
	return nil, fmt.Errorf("%w: body CANDIDATE_SECRET_PAYLOAD123", ErrRateLimited)
}

func (leakyRateLimitedProvider) CountTokens(string) int { return 0 }

// Stream setup failures mark the op span errored; stream consumption itself
// is covered by the WS turn spans (plan batch E).
func TestChatStreamSetupFailureMarksSpan(t *testing.T) {
	exp := &memExporter{}
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exp))
	prev := otelSetTracer(tp)
	t.Cleanup(func() { prev() })

	primary := failingStreamProvider{}
	client := NewClient(primary, nil, nil, 2)
	if _, err := client.ChatStream(context.Background(), ChatRequest{Model: "m"}); err == nil {
		t.Fatal("expected failure")
	}

	found := false
	for _, s := range exp.spans {
		if s.Name() == "llm.chat_stream" {
			found = true
			if s.Status().Code != codes.Error {
				t.Fatal("op span not marked errored")
			}
		}
	}
	if !found {
		t.Fatal("llm.chat_stream span missing")
	}
}

type failingStreamProvider struct{}

func (failingStreamProvider) ChatStream(context.Context, ChatRequest) (<-chan string, error) {
	return nil, ErrUpstream
}

func (failingStreamProvider) Chat(context.Context, ChatRequest) (*ChatResponse, error) {
	return nil, ErrUpstream
}

func (failingStreamProvider) StructuredOutput(context.Context, StructuredRequest) (any, error) {
	return nil, ErrUpstream
}

func (failingStreamProvider) Embed(context.Context, string) ([]float32, error) {
	return nil, ErrUpstream
}

func (failingStreamProvider) CountTokens(string) int { return 0 }

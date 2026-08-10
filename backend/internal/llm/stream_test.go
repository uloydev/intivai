package llm

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

// mockProvider — deterministic token chunks for streaming contract tests.
type mockProvider struct {
	chunks     []string
	streamErr  error
	chatErr    error
	structured any
}

func (m *mockProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	if m.chatErr != nil {
		return nil, m.chatErr
	}
	return &ChatResponse{Content: join(m.chunks)}, nil
}

func (m *mockProvider) ChatStream(ctx context.Context, req ChatRequest) (<-chan string, error) {
	if m.streamErr != nil {
		return nil, m.streamErr
	}
	ch := make(chan string)
	go func() {
		defer close(ch)
		for _, c := range m.chunks {
			ch <- c
		}
	}()
	return ch, nil
}

func (m *mockProvider) StructuredOutput(ctx context.Context, req StructuredRequest) (any, error) {
	if m.structured != nil {
		return m.structured, nil
	}
	return nil, errors.New("unused")
}
func (m *mockProvider) Embed(ctx context.Context, text string) ([]float32, error) {
	return nil, errors.New("unused")
}
func (m *mockProvider) CountTokens(text string) int { return 0 }

func join(parts []string) string {
	out := ""
	for _, p := range parts {
		out += p
	}
	return out
}

func TestChatStreamDeliversChunksInOrderAndCloses(t *testing.T) {
	mock := &mockProvider{chunks: []string{"Hello", " ", "world", "!"}}
	client := NewClient(mock, nil, 1)

	ch, err := client.ChatStream(context.Background(), ChatRequest{})
	if err != nil {
		t.Fatal(err)
	}
	got := ""
	open := true
	for open {
		chunk, ok := <-ch
		if !ok {
			open = false
			continue
		}
		got += chunk
	}
	if got != "Hello world!" {
		t.Fatalf("stream = %q, want %q", got, "Hello world!")
	}
}

func TestChatStreamFallsBackToSecondary(t *testing.T) {
	primary := &mockProvider{streamErr: fmt.Errorf("%w: upstream down", ErrUpstream)}
	fallback := &mockProvider{chunks: []string{"fallback", "-ok"}}
	client := NewClient(primary, fallback, 1)

	ch, err := client.ChatStream(context.Background(), ChatRequest{})
	if err != nil {
		t.Fatal(err)
	}
	got := joinAll(ch)
	if got != "fallback-ok" {
		t.Fatalf("stream = %q", got)
	}
}

func TestChatStreamPropagatesErrorWithoutFallback(t *testing.T) {
	primary := &mockProvider{streamErr: errors.New("upstream down")}
	client := NewClient(primary, nil, 1)

	if _, err := client.ChatStream(context.Background(), ChatRequest{}); err == nil {
		t.Fatal("expected error")
	}
}

func joinAll(ch <-chan string) string {
	out := ""
	for c := range ch {
		out += c
	}
	return out
}

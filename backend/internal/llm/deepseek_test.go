package llm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDeepSeekChat(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"hello"},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":5}}`))
	}))
	defer srv.Close()

	p := NewDeepSeekProvider("sk-test", srv.URL, "deepseek-chat")
	resp, err := p.Chat(context.Background(), ChatRequest{Messages: []Message{{Role: "user", Content: "hi"}}})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Content != "hello" || resp.Usage.CompletionTokens != 5 {
		t.Fatalf("resp = %+v", resp)
	}
}

func TestDeepSeekChatHTTPErrorMapping(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte("slow"))
	}))
	defer srv.Close()

	p := NewDeepSeekProvider("sk-test", srv.URL, "")
	if _, err := p.Chat(context.Background(), ChatRequest{}); err == nil {
		t.Fatal("expected error")
	}
}

func TestDeepSeekChatEmptyChoices(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"choices":[]}`))
	}))
	defer srv.Close()

	p := NewDeepSeekProvider("sk-test", srv.URL, "")
	if _, err := p.Chat(context.Background(), ChatRequest{}); err == nil {
		t.Fatal("empty choices accepted")
	}
}

func TestDeepSeekChatStream(t *testing.T) {
	chunks := []string{"data: {\"choices\":[{\"delta\":{\"content\":\"Hel\"}}]}\n\n",
		"data: {\"choices\":[{\"delta\":{\"content\":\"lo\"}}]}\n\n",
		"data: [DONE]\n\n"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		for _, c := range chunks {
			_, _ = w.Write([]byte(c))
		}
	}))
	defer srv.Close()

	p := NewDeepSeekProvider("sk-test", srv.URL, "")
	ch, err := p.ChatStream(context.Background(), ChatRequest{})
	if err != nil {
		t.Fatal(err)
	}
	got := ""
	for c := range ch {
		got += c
	}
	if got != "Hello" {
		t.Fatalf("stream = %q", got)
	}
}

func TestDeepSeekChatStreamHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	p := NewDeepSeekProvider("sk-test", srv.URL, "")
	if _, err := p.ChatStream(context.Background(), ChatRequest{}); err == nil {
		t.Fatal("expected error")
	}
}

func TestStructuredOutputParseFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"not json"}}]}`))
	}))
	defer srv.Close()

	p := NewDeepSeekProvider("sk-test", srv.URL, "")
	_, err := p.StructuredOutput(context.Background(), StructuredRequest{Schema: &struct {
		A int `json:"a"`
	}{}})
	if err == nil || !strings.Contains(err.Error(), "structured output parse failed") {
		t.Fatalf("err = %v", err)
	}
}

func TestDeepSeekEmbedUnavailable(t *testing.T) {
	p := NewDeepSeekProvider("sk-test", "", "")
	if _, err := p.Embed(context.Background(), "x"); err == nil {
		t.Fatal("embed should be unimplemented")
	}
}

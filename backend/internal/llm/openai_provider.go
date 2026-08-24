package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/intivai/backend/pkg/metrics"
	"github.com/pkoukk/tiktoken-go"
	"github.com/sony/gobreaker"
)

// OpenAIProvider is an OpenAI-compatible chat provider (raw HTTP, no SDK).
// Works with any endpoint exposing POST /chat/completions with Bearer auth:
// SumoPod, OpenRouter, Together, vLLM, Ollama, etc.
type OpenAIProvider struct {
	name    string
	apiKey  string
	baseURL string
	model   string
	http    *http.Client
	cb      *gobreaker.CircuitBreaker
}

// NewOpenAIProvider creates an OpenAI-compatible provider.
//   - name:  human label for metrics/logs (e.g. "sumopod", "openrouter")
//   - apiKey: Bearer token
//   - baseURL: base URL (no trailing slash); the provider appends /chat/completions
//   - model: model identifier sent in every request body
func NewOpenAIProvider(name, apiKey, baseURL, model string, timeoutSeconds int) *OpenAIProvider {
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	if model == "" {
		model = "gpt-4o-mini"
	}
	if timeoutSeconds <= 0 {
		timeoutSeconds = 60
	}
	cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        name + "API",
		MaxRequests: 5,
		Interval:    60 * time.Second,
		Timeout:     30 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.Requests >= 10 && float64(counts.TotalFailures)/float64(counts.Requests) >= 0.5
		},
	})
	return &OpenAIProvider{
		name:    name,
		apiKey:  apiKey,
		baseURL: strings.TrimSuffix(baseURL, "/"),
		model:   model,
		http:    &http.Client{Timeout: time.Duration(timeoutSeconds) * time.Second},
		cb:      cb,
	}
}

type chatRequest struct {
	Model          string    `json:"model"`
	Messages       []Message `json:"messages"`
	Stream         bool      `json:"stream"`
	ResponseFormat *struct {
		Type string `json:"type"`
	} `json:"response_format,omitempty"`
	Temperature float64 `json:"temperature,omitempty"`
	MaxTokens   int     `json:"max_tokens,omitempty"`
}

type chatResponse struct {
	Choices []struct {
		Message      Message `json:"message"`
		FinishReason string  `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

func (p *OpenAIProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	body := chatRequest{
		Model:       or(req.Model, p.model),
		Messages:    req.Messages,
		Stream:      false,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	}
	if req.ResponseFormat == "json_object" {
		body.ResponseFormat = &struct {
			Type string `json:"type"`
		}{Type: "json_object"}
	}

	resp, err := p.do(ctx, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, statusError(resp)
	}

	var out chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode chat response: %w", err)
	}
	if len(out.Choices) == 0 {
		return nil, errors.New("empty choices in chat response")
	}
	respObj := &ChatResponse{
		Content:      out.Choices[0].Message.Content,
		FinishReason: out.Choices[0].FinishReason,
		Usage: Usage{
			PromptTokens:     out.Usage.PromptTokens,
			CompletionTokens: out.Usage.CompletionTokens,
		},
	}

	metrics.LLMTokensTotal.WithLabelValues(p.model, "prompt").Add(float64(respObj.Usage.PromptTokens))
	metrics.LLMTokensTotal.WithLabelValues(p.model, "completion").Add(float64(respObj.Usage.CompletionTokens))

	return respObj, nil
}

func (p *OpenAIProvider) ChatStream(ctx context.Context, req ChatRequest) (<-chan string, error) {
	body := chatRequest{
		Model:    or(req.Model, p.model),
		Messages: req.Messages,
		Stream:   true,
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal streaming chat request: %w", err)
	}

	hreq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/chat/completions", bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	p.setHeaders(hreq)

	res, err := p.cb.Execute(func() (any, error) {
		r, err := p.http.Do(hreq)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil, nil
			}
			return nil, err
		}
		return r, nil
	})
	if res == nil {
		// A nil response is also returned when the caller canceled its context;
		// transport-specific cancellation details are intentionally not exposed.
		return nil, context.Canceled
	}
	if err != nil {
		if errors.Is(err, gobreaker.ErrOpenState) || errors.Is(err, gobreaker.ErrTooManyRequests) {
			return nil, fmt.Errorf("%w: circuit breaker open", ErrUpstream)
		}
		return nil, err
	}
	resp := res.(*http.Response)
	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		return nil, statusError(resp)
	}

	ch := make(chan string)
	go func() {
		defer close(ch)
		defer resp.Body.Close()
		scanner := bufio.NewScanner(resp.Body)
		scanner.Buffer(make([]byte, 64*1024), 64*1024)
		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(line, "data:") {
				continue
			}
			payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if payload == "[DONE]" {
				return
			}
			var chunk struct {
				Choices []struct {
					Delta struct {
						Content string `json:"content"`
					} `json:"delta"`
				} `json:"choices"`
			}
			if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
				continue
			}
			if len(chunk.Choices) > 0 {
				ch <- chunk.Choices[0].Delta.Content
			}
		}
	}()
	return ch, nil
}

func (p *OpenAIProvider) StructuredOutput(ctx context.Context, req StructuredRequest) (any, error) {
	resp, err := p.Chat(ctx, ChatRequest{
		Model: or(req.Model, p.model),
		Messages: []Message{
			{Role: "system", Content: req.System},
			{Role: "user", Content: req.User},
		},
		ResponseFormat: "json_object",
	})
	if err != nil {
		return nil, err
	}
	raw := stripMarkdownFences(resp.Content)
	if req.Schema != nil {
		if err := json.Unmarshal([]byte(raw), req.Schema); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrStructuredParse, err)
		}
		return req.Schema, nil
	}
	var out any
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrStructuredParse, err)
	}
	return out, nil
}

// stripMarkdownFences removes ```json ... ``` wrapping that some LLM providers
// return despite ResponseFormat: json_object. Idempotent on already-clean JSON.
func stripMarkdownFences(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		// strip opening fence (```json or ```JSON or just ```)
		if idx := strings.Index(s, "\n"); idx != -1 {
			s = s[idx+1:]
		}
		// strip closing fence
		if idx := strings.LastIndex(s, "```"); idx != -1 {
			s = s[:idx]
		}
		s = strings.TrimSpace(s)
	}
	return s
}

func (p *OpenAIProvider) Embed(ctx context.Context, text string) ([]float32, error) {
	return nil, errors.New("embedding not implemented: use local fastembed adapter")
}

func (p *OpenAIProvider) CountTokens(text string) int {
	tke, err := tiktoken.GetEncoding("cl100k_base")
	if err != nil {
		return len(strings.Fields(text))
	}
	return len(tke.Encode(text, nil, nil))
}

func (p *OpenAIProvider) do(ctx context.Context, body chatRequest) (*http.Response, error) {
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal chat request: %w", err)
	}
	hreq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/chat/completions", bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	p.setHeaders(hreq)
	res, err := p.cb.Execute(func() (any, error) {
		r, err := p.http.Do(hreq)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil, nil
			}
			return nil, err
		}
		return r, nil
	})
	if res == nil {
		return nil, context.Canceled
	}
	if err != nil {
		if errors.Is(err, gobreaker.ErrOpenState) || errors.Is(err, gobreaker.ErrTooManyRequests) {
			return nil, fmt.Errorf("%w: circuit breaker open", ErrUpstream)
		}
		return nil, err
	}
	return res.(*http.Response), nil
}

func (p *OpenAIProvider) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
}

func statusError(resp *http.Response) error {
	b, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		switch {
		case resp.StatusCode == http.StatusTooManyRequests:
			return fmt.Errorf("%w: read response body: %v", ErrRateLimited, err)
		case resp.StatusCode >= 500:
			return fmt.Errorf("%w: read response body: %v", ErrUpstream, err)
		default:
			return fmt.Errorf("llm api %d: read response body: %v", resp.StatusCode, err)
		}
	}
	switch {
	case resp.StatusCode == http.StatusTooManyRequests:
		return fmt.Errorf("%w: %s", ErrRateLimited, strings.TrimSpace(string(b)))
	case resp.StatusCode >= 500:
		return fmt.Errorf("%w: %d %s", ErrUpstream, resp.StatusCode, strings.TrimSpace(string(b)))
	default:
		return fmt.Errorf("llm api %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
}

func or(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

var _ Provider = (*OpenAIProvider)(nil)

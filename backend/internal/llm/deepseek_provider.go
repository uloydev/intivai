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

// DeepSeekProvider — OpenAI-compatible API (api.deepseek.com/v1), model deepseek-chat.
type DeepSeekProvider struct {
	apiKey  string
	baseURL string
	model   string
	http    *http.Client
	cb      *gobreaker.CircuitBreaker
}

func NewDeepSeekProvider(apiKey, baseURL, model string) *DeepSeekProvider {
	if baseURL == "" {
		baseURL = "https://api.deepseek.com/v1"
	}
	if model == "" {
		model = "deepseek-chat"
	}
	cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        "DeepSeekAPI",
		MaxRequests: 5,
		Interval:    60 * time.Second,
		Timeout:     30 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.Requests >= 10 && float64(counts.TotalFailures)/float64(counts.Requests) >= 0.5
		},
	})
	return &DeepSeekProvider{
		apiKey:  apiKey,
		baseURL: strings.TrimSuffix(baseURL, "/"),
		model:   model,
		http:    &http.Client{Timeout: 60 * time.Second},
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

func (p *DeepSeekProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
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

// ChatStream streams SSE tokens. The channel is closed on completion/error.
func (p *DeepSeekProvider) ChatStream(ctx context.Context, req ChatRequest) (<-chan string, error) {
	body := chatRequest{
		Model:    or(req.Model, p.model),
		Messages: req.Messages,
		Stream:   true,
	}
	raw, _ := json.Marshal(body)

	hreq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/chat/completions", bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	p.setHeaders(hreq)

	res, err := p.cb.Execute(func() (any, error) {
		r, err := p.http.Do(hreq)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil, nil // success for the breaker; caller maps nil → Canceled
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

// StructuredOutput requests JSON mode and validates it parses into Schema.
func (p *DeepSeekProvider) StructuredOutput(ctx context.Context, req StructuredRequest) (any, error) {
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
	if req.Schema != nil {
		if err := json.Unmarshal([]byte(resp.Content), req.Schema); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrStructuredParse, err)
		}
		return req.Schema, nil
	}
	var out any
	if err := json.Unmarshal([]byte(resp.Content), &out); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrStructuredParse, err)
	}
	return out, nil
}

// Embed — TODO(M2): local fastembed (bge-small, 384 dims). DeepSeek has no
// public embedding API; this stays unimplemented until the fastembed adapter lands.
func (p *DeepSeekProvider) Embed(ctx context.Context, text string) ([]float32, error) {
	return nil, errors.New("embedding not implemented: use local fastembed adapter (M2)")
}

// CountTokens approximates the DeepSeek tokenizer with cl100k_base (5-10% drift).
func (p *DeepSeekProvider) CountTokens(text string) int {
	tke, err := tiktoken.GetEncoding("cl100k_base")
	if err != nil {
		return len(strings.Fields(text))
	}
	return len(tke.Encode(text, nil, nil))
}

// The wrapped fn returns (nil, nil) on client cancellation so gobreaker
// records a SUCCESS; the caller turns the nil result back into
// context.Canceled — repeated chat disconnects must not trip the breaker.

func (p *DeepSeekProvider) do(ctx context.Context, body chatRequest) (*http.Response, error) {
	raw, _ := json.Marshal(body)
	hreq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/chat/completions", bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	p.setHeaders(hreq)
	res, err := p.cb.Execute(func() (any, error) {
		r, err := p.http.Do(hreq)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil, nil // success for the breaker; caller maps nil → Canceled
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

func (p *DeepSeekProvider) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
}

func statusError(resp *http.Response) error {
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
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

var _ Provider = (*DeepSeekProvider)(nil)

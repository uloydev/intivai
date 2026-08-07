package llm

import (
	"context"
)

// ChatRequest is provider-agnostic.
type ChatRequest struct {
	Model          string
	Messages       []Message
	ResponseFormat string // "json_object" | ""
	Temperature    float64
	MaxTokens      int
}

type Message struct {
	Role    string `json:"role"` // system | user | assistant
	Content string `json:"content"`
}

type ChatResponse struct {
	Content      string
	FinishReason string
	Usage        Usage
}

type Usage struct {
	PromptTokens     int
	CompletionTokens int
}

type StructuredRequest struct {
	Model  string
	System string
	User   string
	Schema any // Go struct used for json_object validation
}

// Provider — ONE port, every context (cv extraction, interview chat,
// evaluation) uses it. Contexts never define their own LLMProvider.
type Provider interface {
	Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)
	ChatStream(ctx context.Context, req ChatRequest) (<-chan string, error)
	StructuredOutput(ctx context.Context, req StructuredRequest) (any, error)
	Embed(ctx context.Context, text string) ([]float32, error)
	CountTokens(text string) int
}

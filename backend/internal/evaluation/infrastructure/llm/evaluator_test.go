package llm

import (
	"context"
	"errors"
	"testing"

	evaldomain "github.com/intivai/backend/internal/evaluation/domain"
	"github.com/intivai/backend/internal/llm"
)

type mockEvalLLM struct {
	out any
	err error
}

func (m mockEvalLLM) Chat(ctx context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
	return nil, errors.New("unused")
}
func (m mockEvalLLM) ChatStream(ctx context.Context, req llm.ChatRequest) (<-chan string, error) {
	return nil, errors.New("unused")
}
func (m mockEvalLLM) StructuredOutput(ctx context.Context, req llm.StructuredRequest) (any, error) {
	return m.out, m.err
}
func (m mockEvalLLM) Embed(ctx context.Context, text string) ([]float32, error) {
	return nil, errors.New("unused")
}
func (m mockEvalLLM) CountTokens(text string) int { return 0 }

func TestEvaluateValidTranscript(t *testing.T) {
	e := NewEvaluator(mockEvalLLM{out: map[string]any{
		"per_question": []any{
			map[string]any{"question_idx": 1, "category": "technical", "score": 80.0, "rationale": "solid", "strengths": []any{"clear"}, "weaknesses": []any{}},
			map[string]any{"question_idx": 2, "category": "communication", "score": 60.0, "rationale": "brief", "strengths": []any{}, "weaknesses": []any{"short"}},
		},
		"strengths":      []any{"Go"},
		"weaknesses":     []any{"cloud"},
		"recommendation": "proceed",
	}})

	r, err := e.Evaluate(context.Background(), []TranscriptPair{
		{Idx: 1, Category: "technical", Question: "q1", Answer: "a1"},
		{Idx: 2, Category: "communication", Question: "q2", Answer: "a2"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if r.OverallScore != 80*0.4+60*0.2 { // technical .4, communication .2
		t.Fatalf("overall = %f", r.OverallScore)
	}
	if r.Dimensions["technical"].Score != 80 {
		t.Fatalf("technical dim = %f", r.Dimensions["technical"].Score)
	}
	if r.Recommendation != "" || len(r.Strengths) != 0 {
		t.Fatalf("LLM-supplied fields must not leak into the domain report: %+v", r)
	}
}

func TestEvaluateLLMFailure(t *testing.T) {
	e := NewEvaluator(mockEvalLLM{err: errors.New("llm down")})
	if _, err := e.Evaluate(context.Background(), nil); err == nil {
		t.Fatal("llm failure accepted")
	}
}

func TestEvaluateEmptyScoresRejected(t *testing.T) {
	e := NewEvaluator(mockEvalLLM{out: map[string]any{
		"per_question":   []any{},
		"strengths":      []any{},
		"weaknesses":     []any{},
		"recommendation": "proceed",
	}})
	if _, err := e.Evaluate(context.Background(), []TranscriptPair{{Idx: 1}}); err == nil {
		t.Fatal("empty per-question accepted")
	}
}

var _ = evaldomain.Report{}

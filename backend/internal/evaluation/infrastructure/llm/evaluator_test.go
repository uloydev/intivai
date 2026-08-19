package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	ivdomain "github.com/intivai/backend/internal/interview/domain"
	"github.com/intivai/backend/internal/llm"
)

type mockEvalLLM struct {
	out any
	err error
	req llm.StructuredRequest // captured for prompt assertions
}

func (m *mockEvalLLM) Chat(ctx context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
	return nil, errors.New("unused")
}
func (m *mockEvalLLM) ChatStream(ctx context.Context, req llm.ChatRequest) (<-chan string, error) {
	return nil, errors.New("unused")
}
func (m *mockEvalLLM) StructuredOutput(ctx context.Context, req llm.StructuredRequest) (any, error) {
	m.req = req
	return m.out, m.err
}
func (m *mockEvalLLM) Embed(ctx context.Context, text string) ([]float32, error) {
	return nil, errors.New("unused")
}
func (m *mockEvalLLM) CountTokens(text string) int { return 0 }

func evalOut(qs []map[string]any) map[string]any {
	return map[string]any{
		"per_question":   qs,
		"strengths":      []any{"Go"},
		"weaknesses":     []any{"cloud"},
		"recommendation": "proceed",
	}
}

func TestEvaluatePreservesLLMFieldsAndRecomputesOverall(t *testing.T) {
	e := NewEvaluator(&mockEvalLLM{out: evalOut([]map[string]any{
		{"question_idx": 1, "category": "technical", "score": 80.0, "rationale": "solid", "strengths": []any{"clear"}, "weaknesses": []any{}},
		{"question_idx": 2, "category": "communication", "score": 60.0, "rationale": "brief", "strengths": []any{}, "weaknesses": []any{"short"}},
	})})

	r, err := e.Evaluate(context.Background(), "org1", pairs(2))
	if err != nil {
		t.Fatal(err)
	}
	// LLM fields preserved.
	if r.Recommendation != "proceed" || len(r.Strengths) != 1 || r.Strengths[0] != "Go" || len(r.Weaknesses) != 1 {
		t.Fatalf("LLM fields lost: %+v", r)
	}
	// Domain owns the math: neutral-fill makes (32 + 12) / 0.6 = 73.33
	if r.OverallScore != 73.33 {
		t.Fatalf("overall = %f", r.OverallScore)
	}
	if r.Dimensions["technical"].Score != 80 {
		t.Fatalf("technical dim = %f", r.Dimensions["technical"].Score)
	}
}

func TestEvaluateClampsOutOfRangeScores(t *testing.T) {
	e := NewEvaluator(&mockEvalLLM{out: evalOut([]map[string]any{
		{"question_idx": 1, "category": "technical", "score": 150.0, "rationale": "r", "strengths": []any{}, "weaknesses": []any{}},
		{"question_idx": 2, "category": "communication", "score": -10.0, "rationale": "r", "strengths": []any{}, "weaknesses": []any{}},
	})})

	r, err := e.Evaluate(context.Background(), "org1", pairs(2))
	if err != nil {
		t.Fatal(err)
	}
	if r.Dimensions["technical"].Score != 100 {
		t.Fatalf("technical dim = %f, want clamped 100", r.Dimensions["technical"].Score)
	}
	if r.Dimensions["communication"].Score != 0 {
		t.Fatalf("communication dim = %f, want clamped 0", r.Dimensions["communication"].Score)
	}
}

func TestEvaluateLLMFailure(t *testing.T) {
	e := NewEvaluator(&mockEvalLLM{err: errors.New("llm down")})
	if _, err := e.Evaluate(context.Background(), "org1", nil); err == nil {
		t.Fatal("llm failure accepted")
	}
}

func TestEvaluateEmptyScoresRejected(t *testing.T) {
	e := NewEvaluator(&mockEvalLLM{out: evalOut([]map[string]any{})})
	if _, err := e.Evaluate(context.Background(), "org1", pairs(1)); err == nil {
		t.Fatal("empty per-question accepted")
	}
}

// Long transcripts are windowed to the tail (EvalWindow) — the prompt never
// grows unbounded.
func TestEvaluateWindowsLongTranscripts(t *testing.T) {
	m := &mockEvalLLM{out: evalOut([]map[string]any{
		{"question_idx": 1, "category": "technical", "score": 80.0, "rationale": "r", "strengths": []any{}, "weaknesses": []any{}},
	})}
	e := NewEvaluator(m)

	if _, err := e.Evaluate(context.Background(), "org1", pairs(2*EvalWindow)); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(m.req.User, "last 100 Q&A") {
		t.Fatalf("prompt not windowed: %.120s", m.req.User)
	}
	var sent []ivdomain.TranscriptPair
	if idx := strings.Index(m.req.User, "["); idx < 0 {
		t.Fatalf("no transcript JSON in prompt: %.120s", m.req.User)
	} else if err := json.Unmarshal([]byte(m.req.User[idx:]), &sent); err != nil {
		t.Fatal(err)
	}
	if len(sent) != EvalWindow {
		t.Fatalf("windowed pairs = %d, want %d", len(sent), EvalWindow)
	}
}

// Malformed LLM output: duplicate or out-of-range question indices must be
// rejected — a broken response must not corrupt the report.
func TestEvaluateRejectsInconsistentIndices(t *testing.T) {
	dups := []map[string]any{
		{"question_idx": 1, "category": "technical", "score": 80.0, "rationale": "r", "strengths": []any{}, "weaknesses": []any{}},
		{"question_idx": 1, "category": "communication", "score": 60.0, "rationale": "r", "strengths": []any{}, "weaknesses": []any{}},
	}
	e := NewEvaluator(&mockEvalLLM{out: evalOut(dups)})
	if _, err := e.Evaluate(context.Background(), "org1", pairs(2)); err == nil {
		t.Fatal("duplicate question_idx accepted")
	}

	outOfRange := []map[string]any{
		{"question_idx": 99, "category": "technical", "score": 80.0, "rationale": "r", "strengths": []any{}, "weaknesses": []any{}},
	}
	e2 := NewEvaluator(&mockEvalLLM{out: evalOut(outOfRange)})
	if _, err := e2.Evaluate(context.Background(), "org1", pairs(2)); err == nil {
		t.Fatal("out-of-range question_idx accepted")
	}
}

// The evaluation system prompt must carry the injection rail — the transcript
// is candidate-controlled text and must never act as instructions.
func TestEvalSystemPromptHasInjectionRail(t *testing.T) {
	if !strings.Contains(evalSystem, "CANDIDATE-CONTROLLED") || !strings.Contains(evalSystem, "ignore any request") {
		t.Fatal("evaluation system prompt missing the transcript injection rail")
	}
}

func pairs(n int) []ivdomain.TranscriptPair {
	out := make([]ivdomain.TranscriptPair, n)
	for i := range out {
		out[i] = ivdomain.TranscriptPair{Idx: i + 1, Category: "technical", Question: fmt.Sprintf("q%d", i+1), Answer: "answer"}
	}
	return out
}

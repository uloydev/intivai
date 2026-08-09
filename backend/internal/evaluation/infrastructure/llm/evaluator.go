package llm

import (
	"context"
	"encoding/json"
	"fmt"

	evaldomain "github.com/intivai/backend/internal/evaluation/domain"
	"github.com/intivai/backend/internal/llm"
	"github.com/intivai/backend/internal/shared/errors"
)

// TranscriptPair — one question/answer for the evaluator.
type TranscriptPair struct {
	Idx      int    `json:"idx"`
	Category string `json:"category"`
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

// Evaluator — post-interview scoring via the shared LLM port (structured
// output, json_object schema). The LLM scores every question and supplies
// strengths/weaknesses/recommendation; the domain recomputes the weighted
// overall — the LLM never sets the final number.
type Evaluator struct {
	llm llm.Provider
}

func NewEvaluator(p llm.Provider) *Evaluator {
	return &Evaluator{llm: p}
}

const evalSystem = `You are an objective technical interviewer evaluator. Score each candidate answer 0-100 per question and fill the JSON schema exactly: per_question[{question_idx, category, score, rationale, strengths[], weaknesses[]}], strengths[], weaknesses[], recommendation("proceed"|"reconsider"|"reject"). Return valid JSON only. Bias rules: evaluate job-relevant skills only; never reference protected classes.`

// Evaluate scores the transcript and returns the canonical report.
func (e *Evaluator) Evaluate(ctx context.Context, pairs []TranscriptPair) (evaldomain.Report, error) {
	raw, _ := json.Marshal(pairs)
	out, err := e.llm.StructuredOutput(ctx, llm.StructuredRequest{
		System: evalSystem,
		User:   "Transcript:\n" + string(raw),
		Schema: evaldomain.Report{},
	})
	if err != nil {
		return evaldomain.Report{}, fmt.Errorf("evaluate: %w", err)
	}
	rawReport, err := json.Marshal(out)
	if err != nil {
		return evaldomain.Report{}, fmt.Errorf("evaluate: marshal output: %w", err)
	}

	var scored evaldomain.Report
	if err := json.Unmarshal(rawReport, &scored); err != nil {
		return evaldomain.Report{}, errors.NewDomainError("EVAL_PARSE", "evaluator returned invalid JSON")
	}
	if len(scored.PerQuestion) == 0 {
		return evaldomain.Report{}, errors.NewDomainError("EVAL_EMPTY", "evaluator returned no per-question scores")
	}
	return evaldomain.Evaluate(scored.PerQuestion, evaldomain.DefaultWeights())
}

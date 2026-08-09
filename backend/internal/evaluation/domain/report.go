package domain

import (
	"fmt"
	"math"

	"github.com/intivai/backend/internal/shared/errors"
)

// Report — canonical evaluation schema (Phases doc §Phase 4, single source
// of truth for Research §2/§5). Persisted on interviews.evaluation JSONB.
type Report struct {
	OverallScore   float64              `json:"overall_score"`
	Dimensions     map[string]Dimension `json:"dimensions"`
	PerQuestion    []QuestionScore      `json:"per_question"`
	Strengths      []string             `json:"strengths"`
	Weaknesses     []string             `json:"weaknesses"`
	Recommendation string               `json:"recommendation"`
}

type Dimension struct {
	Score  float64 `json:"score"`
	Weight float64 `json:"weight"`
}

// QuestionScore — per-question LLM score; Category drives the dimension
// roll-up (LLM supplies it from the question bank categories).
type QuestionScore struct {
	QuestionIdx int      `json:"question_idx"`
	Score       float64  `json:"score"`
	Rationale   string   `json:"rationale"`
	Strengths   []string `json:"strengths"`
	Weaknesses  []string `json:"weaknesses"`
	Category    string   `json:"category"`
}

// DefaultWeights — canonical dimension weights.
func DefaultWeights() map[string]float64 {
	return map[string]float64{
		"technical":       0.4,
		"communication":   0.2,
		"problem_solving": 0.25,
		"culture_fit":     0.15,
	}
}

// Evaluate aggregates per-question scores into the final report: dimension
// score = mean of its questions' scores (by category), overall = weighted
// sum. Weights must sum to 1.0. Empty transcript → zeroed report.
func Evaluate(perQuestion []QuestionScore, weights map[string]float64) (Report, error) {
	sum := 0.0
	for _, w := range weights {
		sum += w
	}
	if math.Abs(sum-1) > 1e-9 {
		return Report{}, errors.NewDomainError("EVAL_WEIGHTS", fmt.Sprintf("weights must sum to 1.0, got %f", sum))
	}

	r := Report{
		Dimensions:  make(map[string]Dimension, len(weights)),
		PerQuestion: perQuestion,
	}
	for name, w := range weights {
		r.Dimensions[name] = Dimension{Weight: w}
	}

	counts := map[string]int{}
	for _, q := range perQuestion {
		d, ok := r.Dimensions[q.Category]
		if !ok {
			continue // unknown category: not part of the report
		}
		counts[q.Category]++
		d.Score = (d.Score*float64(counts[q.Category]-1) + q.Score) / float64(counts[q.Category])
		r.Dimensions[q.Category] = d
	}

	total := 0.0
	for _, d := range r.Dimensions {
		total += clamp(d.Score, 0, 100) * d.Weight
	}
	r.OverallScore = math.Round(total*100) / 100
	return r, nil
}

func clamp(v, lo, hi float64) float64 {
	if v > hi {
		return hi
	}
	if v < lo {
		return lo
	}
	return v
}

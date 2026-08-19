package domain

import (
	"fmt"
	"math"

	"github.com/intivai/backend/internal/shared/errors"
)

// Report — canonical evaluation schema (Phases doc §Phase 4, single source
// of truth for Research §2/§5). Persisted on interviews.evaluation JSONB.
// Recommendation verdicts — the only values the evaluator may emit; the LLM's
// free-text verdict is constrained to these at the adapter boundary so one
// hallucinated run cannot become a hire/reject decision.
type Recommendation string

const (
	RecommendationProceed    Recommendation = "proceed"
	RecommendationReconsider Recommendation = "reconsider"
	RecommendationReject     Recommendation = "reject"
)

// ValidRecommendation reports whether v is a known verdict.
func ValidRecommendation(v string) bool {
	switch Recommendation(v) {
	case RecommendationProceed, RecommendationReconsider, RecommendationReject:
		return true
	}
	return false
}

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
	Quotes      []string `json:"quotes"`
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
	weightSum := 0.0
	for name, d := range r.Dimensions {
		if counts[name] > 0 {
			total += clamp(d.Score, 0, 100) * d.Weight
			weightSum += d.Weight
		}
	}

	// Neutral-fill: if there's any valid score, missing dimensions take the average of what WAS scored
	// (so they neither penalize nor artificially boost the overall outcome).
	var averageScore float64
	if weightSum > 0 {
		averageScore = total / weightSum
	} else {
		averageScore = 0 // edge case: no questions mapped to any known dimension
	}

	for name, d := range r.Dimensions {
		if counts[name] == 0 {
			d.Score = averageScore
			r.Dimensions[name] = d
		}
	}

	r.OverallScore = math.Round(averageScore*100) / 100
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

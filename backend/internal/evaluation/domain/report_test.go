package domain

import (
	"math"
	"testing"
)

func TestEvaluateWeightsMustSumToOne(t *testing.T) {
	_, err := Evaluate(nil, map[string]float64{"technical": 0.5})
	if err == nil {
		t.Fatal("weights summing to 0.5 accepted")
	}
}

func TestEvaluateEmptyTranscript(t *testing.T) {
	r, err := Evaluate(nil, DefaultWeights())
	if err != nil {
		t.Fatal(err)
	}
	if r.OverallScore != 0 || len(r.Dimensions) != 4 || r.Recommendation != "" {
		t.Fatalf("empty transcript report = %+v", r)
	}
}

func TestEvaluateSingleAnswer(t *testing.T) {
	r, err := Evaluate([]QuestionScore{
		{QuestionIdx: 1, Score: 80, Category: "technical"},
	}, DefaultWeights())
	if err != nil {
		t.Fatal(err)
	}
	want := 80 * 0.4 // technical weight
	if math.Abs(r.OverallScore-want) > 1e-9 {
		t.Fatalf("overall = %f, want %f", r.OverallScore, want)
	}
	if r.Dimensions["technical"].Score != 80 {
		t.Fatalf("technical dim = %f, want 80", r.Dimensions["technical"].Score)
	}
}

func TestEvaluateAveragesPerDimension(t *testing.T) {
	r, err := Evaluate([]QuestionScore{
		{QuestionIdx: 1, Score: 100, Category: "technical"},
		{QuestionIdx: 2, Score: 60, Category: "technical"},
	}, DefaultWeights())
	if err != nil {
		t.Fatal(err)
	}
	if r.Dimensions["technical"].Score != 80 {
		t.Fatalf("technical dim = %f, want avg 80", r.Dimensions["technical"].Score)
	}
}

func TestEvaluateLongInterview(t *testing.T) {
	qs := make([]QuestionScore, 50)
	for i := range qs {
		qs[i] = QuestionScore{QuestionIdx: i + 1, Score: 75, Category: "technical"}
	}
	r, err := Evaluate(qs, DefaultWeights())
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(r.OverallScore-30) > 1e-9 { // 75 × 0.4
		t.Fatalf("overall = %f, want 30", r.OverallScore)
	}
}

func TestEvaluateUnknownCategoryIgnored(t *testing.T) {
	r, err := Evaluate([]QuestionScore{
		{QuestionIdx: 1, Score: 90, Category: "mystery"},
	}, DefaultWeights())
	if err != nil {
		t.Fatal(err)
	}
	for name, d := range r.Dimensions {
		if d.Score != 0 {
			t.Fatalf("dim %s = %f, want 0 (unknown category)", name, d.Score)
		}
	}
}

func TestEvaluateClampsOverall(t *testing.T) {
	r, err := Evaluate([]QuestionScore{
		{QuestionIdx: 1, Score: 150, Category: "technical"},
	}, DefaultWeights())
	if err != nil {
		t.Fatal(err)
	}
	if r.OverallScore > 100 {
		t.Fatalf("overall = %f, want <= 100", r.OverallScore)
	}
}

func TestDefaultWeightsSumToOne(t *testing.T) {
	sum := 0.0
	for _, w := range DefaultWeights() {
		sum += w
	}
	if math.Abs(sum-1) > 1e-9 {
		t.Fatalf("weights sum = %f, want 1", sum)
	}
}

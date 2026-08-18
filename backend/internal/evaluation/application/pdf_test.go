package application

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestGeneratePDFReport(t *testing.T) {
	eval := evaluationParsed{
		OverallScore: 88,
		Dimensions: map[string]struct {
			Score  int     `json:"score"`
			Weight float64 `json:"weight"`
		}{
			"technical":     {Score: 90, Weight: 0.4},
			"communication": {Score: 85, Weight: 0.3},
		},
		Strengths:      []string{"Strong Go concurrency skills", "Clear explanations"},
		Weaknesses:     []string{"Could optimize memory allocations"},
		Recommendation: "proceed",
		PerQuestion: []struct {
			QuestionIdx int      `json:"question_idx"`
			Score       int      `json:"score"`
			Rationale   string   `json:"rationale"`
			Strengths   []string `json:"strengths"`
			Weaknesses  []string `json:"weaknesses"`
		}{
			{
				QuestionIdx: 1,
				Score:       90,
				Rationale:   "Solid understanding of CSP channels",
				Strengths:   []string{"Goroutines"},
			},
		},
	}
	evalBytes, _ := json.Marshal(eval)

	detail := &InterviewDetail{
		InterviewID: uuid.New(),
		Status:      "completed",
		CreatedAt:   time.Now(),
		Candidate: &CandidateDTO{
			ID:    uuid.New(),
			Name:  "Jane Doe",
			Email: "jane@example.com",
		},
		Job: &JobDTO{
			ID:    uuid.New(),
			Title: "Senior Go Engineer",
		},
		Questions: []QuestionDTO{
			{Idx: 1, Content: "Explain Goroutines vs Threads"},
		},
		Answers: []AnswerDTO{
			{Idx: 1, Content: "Goroutines are multiplexed onto OS threads by the Go runtime scheduler."},
		},
		Evaluation: evalBytes,
	}

	pdfBytes, err := generatePDFReport(detail)
	if err != nil {
		t.Fatalf("generatePDFReport failed: %v", err)
	}
	if len(pdfBytes) == 0 {
		t.Fatal("expected non-empty PDF bytes")
	}
	if string(pdfBytes[:4]) != "%PDF" {
		t.Fatalf("expected PDF magic bytes, got %q", string(pdfBytes[:4]))
	}
}

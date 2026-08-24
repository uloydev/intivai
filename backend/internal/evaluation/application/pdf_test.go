package application

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ledongthuc/pdf"

	ivdomain "github.com/intivai/backend/internal/interview/domain"
)

func extractPDFText(t *testing.T, pdfBytes []byte) string {
	t.Helper()
	reader, err := pdf.NewReader(bytes.NewReader(pdfBytes), int64(len(pdfBytes)))
	if err != nil {
		t.Fatalf("open pdf: %v", err)
	}
	var sb strings.Builder
	for i := 1; i <= reader.NumPage(); i++ {
		pageText, err := reader.Page(i).GetPlainText(nil)
		if err != nil {
			t.Fatalf("extract page %d text: %v", i, err)
		}
		sb.WriteString(pageText)
	}
	return strings.Join(strings.Fields(sb.String()), " ")
}

func newProctoringPDFDetail(t *testing.T, summary ivdomain.ProctoringSummary) *InterviewDetail {
	t.Helper()
	eval := evaluationParsed{
		OverallScore: 80.0,
		Dimensions: map[string]struct {
			Score  float64 `json:"score"`
			Weight float64 `json:"weight"`
		}{
			"technical": {Score: 80.0, Weight: 1.0},
		},
		Strengths:      []string{"Solid fundamentals"},
		Recommendation: "proceed",
	}
	evalBytes, err := json.Marshal(eval)
	if err != nil {
		t.Fatal(err)
	}
	return &InterviewDetail{
		InterviewID: uuid.New(),
		Status:      "completed",
		CreatedAt:   time.Now(),
		Candidate: &CandidateDTO{
			ID:    uuid.New(),
			Name:  "Jane Doe",
			Email: "jane@example.com",
		},
		Questions:         []QuestionDTO{{Idx: 1, Content: "Explain Goroutines vs Threads"}},
		Answers:           []AnswerDTO{{Idx: 1, Content: "Goroutines are multiplexed onto OS threads by the Go runtime scheduler."}},
		Evaluation:        evalBytes,
		ProctoringSummary: summary,
	}
}

func TestGeneratePDFReport(t *testing.T) {
	eval := evaluationParsed{
		OverallScore: 88.5,
		Dimensions: map[string]struct {
			Score  float64 `json:"score"`
			Weight float64 `json:"weight"`
		}{
			"technical":       {Score: 92.0, Weight: 0.4},
			"problem_solving": {Score: 88.0, Weight: 0.25},
			"communication":   {Score: 85.0, Weight: 0.2},
			"culture_fit":     {Score: 90.0, Weight: 0.15},
		},
		Strengths:      []string{"Strong Go concurrency and goroutine lifecycle understanding", "Clear architectural explanations of distributed consensus"},
		Weaknesses:     []string{"Could optimize heap allocations in hot path"},
		Recommendation: "proceed",
		PerQuestion: []struct {
			QuestionIdx int      `json:"question_idx"`
			Score       float64  `json:"score"`
			Rationale   string   `json:"rationale"`
			Quotes      []string `json:"quotes"`
			Strengths   []string `json:"strengths"`
			Weaknesses  []string `json:"weaknesses"`
			Category    string   `json:"category"`
		}{
			{
				QuestionIdx: 1,
				Score:       92.0,
				Rationale:   "Solid understanding of CSP channels and Go runtime M:N scheduler.",
				Quotes:      []string{"Goroutines are multiplexed onto OS threads by the Go runtime scheduler."},
				Strengths:   []string{"Goroutines", "Scheduler internals"},
				Category:    "Technical Execution",
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

func TestGeneratePDFReport_ChloeDubois(t *testing.T) {
	eval := evaluationParsed{
		OverallScore: 94.0,
		Dimensions: map[string]struct {
			Score  float64 `json:"score"`
			Weight float64 `json:"weight"`
		}{
			"technical":       {Score: 96.0, Weight: 0.40},
			"problem_solving": {Score: 92.0, Weight: 0.25},
			"communication":   {Score: 95.0, Weight: 0.20},
			"culture_fit":     {Score: 90.0, Weight: 0.15},
		},
		Strengths: []string{
			"Comprehensive understanding of browser automation and parallel worker state isolation",
			"Strong grasp of CI/CD gate automation and automated load regression barriers",
		},
		Weaknesses: []string{
			"Could elaborate further on containerized browser cluster scaling benchmarks",
		},
		Recommendation: "proceed",
		PerQuestion: []struct {
			QuestionIdx int      `json:"question_idx"`
			Score       float64  `json:"score"`
			Rationale   string   `json:"rationale"`
			Quotes      []string `json:"quotes"`
			Strengths   []string `json:"strengths"`
			Weaknesses  []string `json:"weaknesses"`
			Category    string   `json:"category"`
		}{
			{
				QuestionIdx: 1,
				Score:       96.0,
				Rationale:   "Clear mastery of test isolation patterns and browser automation best practices.",
				Quotes:      []string{"We use dynamic test database schemas per worker or transaction rollback fixtures"},
				Strengths:   []string{"Playwright worker isolation", "Hermetic fixtures"},
				Category:    "Technical Architecture",
			},
			{
				QuestionIdx: 2,
				Score:       92.0,
				Rationale:   "Demonstrated proactive latency regression guarding via threshold gates in CI.",
				Quotes:      []string{"threshold-gated k6 scripts checking p95 latency under 100 concurrent virtual users"},
				Strengths:   []string{"k6 load testing", "GitHub Actions CI gates"},
				Category:    "Performance Engineering",
			},
		},
	}
	evalBytes, _ := json.Marshal(eval)

	detail := &InterviewDetail{
		InterviewID: uuid.MustParse("d5e6f7a8-b4c5-4d6e-8f7a-8b4c5d6e7f8a"),
		Status:      "completed",
		CreatedAt:   time.Date(2026, 8, 19, 14, 30, 0, 0, time.UTC),
		Candidate: &CandidateDTO{
			ID:    uuid.New(),
			Name:  "Chloe Dubois",
			Email: "chloe.dubois@example.com",
		},
		Job: &JobDTO{
			ID:    uuid.New(),
			Title: "Senior SDET & Test Automation Architect",
		},
		Questions: []QuestionDTO{
			{Idx: 1, Content: "How do you design hermetic Playwright test suites that prevent flaky test runs and isolate shared database state across parallel test workers?", Category: "Technical Architecture"},
			{Idx: 2, Content: "How do you integrate automated k6 performance tests into GitHub Actions CI to prevent latency regressions?", Category: "Performance Engineering"},
		},
		Answers: []AnswerDTO{
			{Idx: 1, Content: "We use dynamic test database schemas per worker or transaction rollback fixtures, along with deterministic seed harnesses and custom locator retry thresholds.", AnsweredAt: time.Now()},
			{Idx: 2, Content: "We run threshold-gated k6 scripts checking p95 latency under 100 concurrent virtual users, automatically blocking PR merge if SLA exceeds 150ms.", AnsweredAt: time.Now()},
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
	if err := os.WriteFile(filepath.Join(t.TempDir(), "zero_telemetry_report.pdf"), pdfBytes, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
}

func TestGeneratePDFReportRejectsMalformedEvaluation(t *testing.T) {
	_, err := generatePDFReport(&InterviewDetail{InterviewID: uuid.New(), Evaluation: []byte(`{"overall_score":`)})
	if err == nil {
		t.Fatal("expected malformed evaluation error")
	}
}

func TestGeneratePDFReportClampsDimensionLayoutAndText(t *testing.T) {
	eval := evaluationParsed{
		OverallScore: 140,
		Dimensions: map[string]struct {
			Score  float64 `json:"score"`
			Weight float64 `json:"weight"`
		}{
			"one": {Score: -10}, "two": {Score: 101}, "three": {Score: 50},
			"four": {Score: 50}, "five": {Score: 50},
		},
	}
	raw, err := json.Marshal(eval)
	if err != nil {
		t.Fatal(err)
	}
	detail := &InterviewDetail{
		InterviewID: uuid.New(),
		Candidate:   &CandidateDTO{Name: strings.Repeat("candidate\n", 30), Email: strings.Repeat("email\t", 30)},
		Questions:   []QuestionDTO{{Idx: 1, Content: strings.Repeat("question ", 30)}},
		Answers:     []AnswerDTO{{Idx: 1, Content: strings.Repeat("answer ", 30)}},
		Evaluation:  raw,
	}
	pdf, err := generatePDFReport(detail)
	if err != nil {
		t.Fatalf("generatePDFReport failed: %v", err)
	}
	if len(pdf) == 0 || string(pdf[:4]) != "%PDF" {
		t.Fatalf("expected valid PDF output")
	}
	if got := cleanPDFText(strings.Repeat("x", 101)); len(got) != 100 {
		t.Fatalf("cleanPDFText length = %d, want 100", len(got))
	}
}

func TestGeneratePDFReportZeroTelemetryStatesNoTelemetry(t *testing.T) {
	detail := newProctoringPDFDetail(t, ivdomain.ProctoringSummary{})
	pdfBytes, err := generatePDFReport(detail)
	if err != nil {
		t.Fatalf("generatePDFReport failed: %v", err)
	}
	text := extractPDFText(t, pdfBytes)
	if !strings.Contains(text, "No telemetry recorded") {
		t.Fatalf("expected PDF to state no telemetry recorded; got: %s", text)
	}
	if strings.Contains(text, "Integrity Score:") {
		t.Fatalf("PDF must not fabricate an integrity score without telemetry; got: %s", text)
	}
	if strings.Contains(text, "Verified Session Authenticity") {
		t.Fatalf("PDF must not claim verified authenticity without telemetry; got: %s", text)
	}
}

func TestGeneratePDFReportWithTelemetryShowsIntegrityScore(t *testing.T) {
	detail := newProctoringPDFDetail(t, ivdomain.ProctoringSummary{
		IntegrityScore:       90,
		RiskLevel:            ivdomain.RiskLow,
		TabSwitchCount:       1,
		TotalAwayDurationSec: 30,
		PasteEventCount:      1,
	})
	pdfBytes, err := generatePDFReport(detail)
	if err != nil {
		t.Fatalf("generatePDFReport failed: %v", err)
	}
	text := extractPDFText(t, pdfBytes)
	for _, want := range []string{
		"Integrity Score: 90 / 100",
		"Risk Level: LOW",
		"Verified Session Authenticity",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected PDF text to contain %q; got: %s", want, text)
		}
	}
}

func TestGeneratePDFReportFlaggedTelemetryKeepsAuditRendering(t *testing.T) {
	detail := newProctoringPDFDetail(t, ivdomain.ProctoringSummary{
		IntegrityScore:    55,
		RiskLevel:         ivdomain.RiskHigh,
		TabSwitchCount:    3,
		AudioAnomalyCount: 1,
		Flags:             []string{"Candidate switched tabs or blurred browser 3 time(s)"},
	})
	pdfBytes, err := generatePDFReport(detail)
	if err != nil {
		t.Fatalf("generatePDFReport failed: %v", err)
	}
	text := extractPDFText(t, pdfBytes)
	for _, want := range []string{
		"Integrity Score: 55 / 100",
		"Risk Level: HIGH",
		"Verified Session Authenticity",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected PDF text to contain %q; got: %s", want, text)
		}
	}
	if strings.Contains(text, "No telemetry recorded") {
		t.Fatalf("flagged session has telemetry; PDF must not claim no telemetry; got: %s", text)
	}
}

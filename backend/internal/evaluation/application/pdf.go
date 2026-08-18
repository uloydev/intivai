package application

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	ivdomain "github.com/intivai/backend/internal/interview/domain"
	"github.com/johnfercher/maroto/v2"
	"github.com/johnfercher/maroto/v2/pkg/components/text"
	"github.com/johnfercher/maroto/v2/pkg/config"
	"github.com/johnfercher/maroto/v2/pkg/consts/align"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/props"
)

type evaluationParsed struct {
	OverallScore int `json:"overall_score"`
	Dimensions   map[string]struct {
		Score  int     `json:"score"`
		Weight float64 `json:"weight"`
	} `json:"dimensions"`
	PerQuestion []struct {
		QuestionIdx int      `json:"question_idx"`
		Score       int      `json:"score"`
		Rationale   string   `json:"rationale"`
		Strengths   []string `json:"strengths"`
		Weaknesses  []string `json:"weaknesses"`
	} `json:"per_question"`
	Strengths      []string `json:"strengths"`
	Weaknesses     []string `json:"weaknesses"`
	Recommendation string   `json:"recommendation"`
}

func generatePDFReport(detail *InterviewDetail) ([]byte, error) {
	cfg := config.NewBuilder().
		WithPageSize("A4").
		WithLeftMargin(10).
		WithRightMargin(10).
		WithTopMargin(15).
		WithBottomMargin(15).
		Build()

	m := maroto.New(cfg)

	m.AddRow(15,
		text.NewCol(12, "Intivai Candidate Evaluation Report", props.Text{
			Family: "arial",
			Style:  fontstyle.Bold,
			Size:   16,
			Align:  align.Center,
		}),
	)
	m.AddRow(5)

	name := "Unknown Candidate"
	if detail.Candidate != nil {
		name = detail.Candidate.Name
	}
	jobTitle := "Unknown Job"
	if detail.Job != nil {
		jobTitle = detail.Job.Title
	}

	m.AddRow(10,
		text.NewCol(6, fmt.Sprintf("Candidate: %s", name), props.Text{Size: 10, Style: fontstyle.Bold}),
		text.NewCol(6, fmt.Sprintf("Job Role: %s", jobTitle), props.Text{Size: 10, Style: fontstyle.Bold}),
	)
	m.AddRow(10,
		text.NewCol(6, fmt.Sprintf("Date: %s", detail.CreatedAt.Format("02 Jan 2006")), props.Text{Size: 10}),
		text.NewCol(6, fmt.Sprintf("Status: %s", detail.Status), props.Text{Size: 10}),
	)
	m.AddRow(5)

	var eval evaluationParsed
	if len(detail.Evaluation) > 0 {
		_ = json.Unmarshal(detail.Evaluation, &eval)
	}

	m.AddRow(12,
		text.NewCol(6, fmt.Sprintf("Overall Score: %d / 100", eval.OverallScore), props.Text{Size: 12, Style: fontstyle.Bold}),
		text.NewCol(6, fmt.Sprintf("Recommendation: %s", eval.Recommendation), props.Text{Size: 12, Style: fontstyle.Bold}),
	)
	m.AddRow(5)

	m.AddRow(10, text.NewCol(12, "Dimensions:", props.Text{Size: 11, Style: fontstyle.Bold}))
	var dimNames []string
	for k := range eval.Dimensions {
		dimNames = append(dimNames, k)
	}
	sort.Strings(dimNames)
	for _, k := range dimNames {
		v := eval.Dimensions[k]
		m.AddRow(6, text.NewCol(12, fmt.Sprintf("  - %s: %d (weight: %.2f)", k, v.Score, v.Weight), props.Text{Size: 9}))
	}
	m.AddRow(5)

	m.AddRow(10, text.NewCol(12, "Strengths:", props.Text{Size: 11, Style: fontstyle.Bold}))
	for _, s := range eval.Strengths {
		m.AddRow(6, text.NewCol(12, fmt.Sprintf("  + %s", s), props.Text{Size: 9}))
	}
	m.AddRow(5)

	m.AddRow(10, text.NewCol(12, "Weaknesses:", props.Text{Size: 11, Style: fontstyle.Bold}))
	for _, w := range eval.Weaknesses {
		m.AddRow(6, text.NewCol(12, fmt.Sprintf("  - %s", w), props.Text{Size: 9}))
	}
	m.AddRow(5)

	// Proctoring & Integrity Audit
	summary := detail.ProctoringSummary
	if summary.IntegrityScore == 0 && len(detail.ProctoringEvents) == 0 {
		summary = ivdomain.DefaultProctoringSummary()
	}

	m.AddRow(10, text.NewCol(12, "Anti-Cheating & Integrity Audit:", props.Text{Size: 11, Style: fontstyle.Bold}))
	m.AddRow(6, text.NewCol(12, fmt.Sprintf("  - Integrity Score: %d / 100 (Risk Tier: %s)", summary.IntegrityScore, strings.ToUpper(string(summary.RiskLevel))), props.Text{Size: 9}))
	m.AddRow(6, text.NewCol(12, fmt.Sprintf("  - Tab Switches: %d | Time Away: %ds | Pastes: %d (Suspicious: %d) | Audio Anomalies: %d",
		summary.TabSwitchCount, summary.TotalAwayDurationSec, summary.PasteEventCount, summary.SuspiciousPasteCount, summary.AudioAnomalyCount), props.Text{Size: 9}))
	if len(summary.Flags) > 0 {
		for _, f := range summary.Flags {
			m.AddRow(6, text.NewCol(12, fmt.Sprintf("  ! Flag: %s", f), props.Text{Size: 9, Style: fontstyle.Italic}))
		}
	} else {
		m.AddRow(6, text.NewCol(12, "  ✓ Spotless Session: Zero integrity anomalies detected", props.Text{Size: 9}))
	}
	m.AddRow(8)

	ansMap := make(map[int]string)
	for _, a := range detail.Answers {
		ansMap[a.Idx] = a.Content
	}
	qScoreMap := make(map[int]int)
	qRatMap := make(map[int]string)
	for _, pq := range eval.PerQuestion {
		qScoreMap[pq.QuestionIdx] = pq.Score
		qRatMap[pq.QuestionIdx] = pq.Rationale
	}

	m.AddRow(10, text.NewCol(12, "Per-Question Breakdown:", props.Text{Size: 12, Style: fontstyle.Bold}))
	m.AddRow(5)

	for _, q := range detail.Questions {
		m.AddRow(10, text.NewCol(12, fmt.Sprintf("Q%d: %s", q.Idx, q.Content), props.Text{Size: 10, Style: fontstyle.Bold}))

		ansText := ansMap[q.Idx]
		if ansText == "" {
			ansText = "(No answer provided)"
		}
		// Allocate 30 for answer text wrapping
		m.AddRow(30, text.NewCol(12, fmt.Sprintf("Answer: %s", ansText), props.Text{Size: 9}))

		scoreText := fmt.Sprintf("Score: %d | Rationale: %s", qScoreMap[q.Idx], qRatMap[q.Idx])
		m.AddRow(15, text.NewCol(12, scoreText, props.Text{Size: 9, Style: fontstyle.Italic}))
		m.AddRow(5)
	}

	doc, err := m.Generate()
	if err != nil {
		return nil, fmt.Errorf("generate pdf: %w", err)
	}

	return doc.GetBytes(), nil
}

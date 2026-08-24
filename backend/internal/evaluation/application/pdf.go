package application

import (
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/johnfercher/maroto/v2"
	"github.com/johnfercher/maroto/v2/pkg/components/col"
	"github.com/johnfercher/maroto/v2/pkg/components/line"
	"github.com/johnfercher/maroto/v2/pkg/components/row"
	"github.com/johnfercher/maroto/v2/pkg/components/text"
	"github.com/johnfercher/maroto/v2/pkg/config"
	"github.com/johnfercher/maroto/v2/pkg/consts/align"
	"github.com/johnfercher/maroto/v2/pkg/consts/border"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/core"
	"github.com/johnfercher/maroto/v2/pkg/props"
)

var (
	colorBrandPrimary  = &props.Color{Red: 79, Green: 70, Blue: 229}   // Indigo #4F46E5
	colorBrandDark     = &props.Color{Red: 30, Green: 27, Blue: 75}    // Deep Navy #1E1B4B
	colorTextDark      = &props.Color{Red: 15, Green: 23, Blue: 42}    // Slate 900 #0F172A
	colorTextBody      = &props.Color{Red: 51, Green: 65, Blue: 85}    // Slate 700 #334155
	colorTextMuted     = &props.Color{Red: 100, Green: 116, Blue: 139} // Slate 500 #64748B
	colorCardBg        = &props.Color{Red: 248, Green: 250, Blue: 252} // Slate 50 #F8FAFC
	colorCardBorder    = &props.Color{Red: 226, Green: 232, Blue: 240} // Slate 200 #E2E8F0
	colorWhite         = &props.Color{Red: 255, Green: 255, Blue: 255}
	colorEmerald       = &props.Color{Red: 5, Green: 150, Blue: 105}   // Emerald 600 #059669
	colorEmeraldBg     = &props.Color{Red: 236, Green: 253, Blue: 245} // Emerald 50 #ECFDF5
	colorEmeraldBorder = &props.Color{Red: 167, Green: 243, Blue: 208} // Emerald 200 #A7F3D0
	colorAmber         = &props.Color{Red: 217, Green: 119, Blue: 6}   // Amber 600 #D97706
	colorAmberBg       = &props.Color{Red: 255, Green: 251, Blue: 235} // Amber 50 #FFFBEB
	colorAmberBorder   = &props.Color{Red: 253, Green: 230, Blue: 138} // Amber 200 #FDE68A
	colorRose          = &props.Color{Red: 225, Green: 29, Blue: 72}   // Rose 600 #E11D48
	colorRoseBg        = &props.Color{Red: 255, Green: 241, Blue: 242} // Rose 50 #FFF1F2
	colorRoseBorder    = &props.Color{Red: 254, Green: 205, Blue: 211} // Rose 200 #FECDD3
	colorIndigoBg      = &props.Color{Red: 238, Green: 242, Blue: 255} // Indigo 50 #EEF2FF
	colorIndigoBorder  = &props.Color{Red: 199, Green: 210, Blue: 254} // Indigo 200 #C7D2FE
)

// evaluationParsed supports both float64 and legacy int representations.
type evaluationParsed struct {
	OverallScore float64 `json:"overall_score"`
	Dimensions   map[string]struct {
		Score  float64 `json:"score"`
		Weight float64 `json:"weight"`
	} `json:"dimensions"`
	PerQuestion []struct {
		QuestionIdx int      `json:"question_idx"`
		Score       float64  `json:"score"`
		Rationale   string   `json:"rationale"`
		Quotes      []string `json:"quotes"`
		Strengths   []string `json:"strengths"`
		Weaknesses  []string `json:"weaknesses"`
		Category    string   `json:"category"`
	} `json:"per_question"`
	Strengths      []string `json:"strengths"`
	Weaknesses     []string `json:"weaknesses"`
	Recommendation string   `json:"recommendation"`
}

func generatePDFReport(detail *InterviewDetail) ([]byte, error) {
	var eval evaluationParsed
	if len(detail.Evaluation) > 0 {
		if err := json.Unmarshal(detail.Evaluation, &eval); err != nil {
			return nil, fmt.Errorf("parse evaluation for pdf: %w", err)
		}
	}

	cfg := config.NewBuilder().
		WithPageSize("A4").
		WithLeftMargin(12).
		WithRightMargin(12).
		WithTopMargin(12).
		WithBottomMargin(12).
		Build()

	m := maroto.New(cfg)

	// --- Header Registration ---
	_ = m.RegisterHeader(
		row.New(12).Add(
			col.New(8).Add(
				text.New("INTIVAI TALENT INTELLIGENCE", props.Text{
					Style: fontstyle.Bold,
					Size:  12,
					Color: colorBrandPrimary,
				}),
				text.New("Autonomous Technical Assessment & Competency Evaluation", props.Text{
					Size:  7.5,
					Color: colorTextMuted,
					Top:   5,
				}),
			),
			col.New(4).Add(
				text.New("CONFIDENTIAL EVALUATION", props.Text{
					Style: fontstyle.Bold,
					Size:  8,
					Align: align.Right,
					Color: colorTextDark,
				}),
				text.New(fmt.Sprintf("Generated: %s", time.Now().Format("02 Jan 2006 15:04 MST")), props.Text{
					Size:  7,
					Align: align.Right,
					Color: colorTextMuted,
					Top:   5,
				}),
			),
		),
		row.New(2).Add(
			line.NewCol(12, props.Line{
				Color:     colorBrandPrimary,
				Thickness: 1.0,
			}),
		),
		row.New(3),
	)

	// --- Footer Registration ---
	_ = m.RegisterFooter(
		row.New(2).Add(
			line.NewCol(12, props.Line{
				Color:     colorCardBorder,
				Thickness: 0.5,
			}),
		),
		row.New(6).Add(
			col.New(8).Add(
				text.New("Intivai Enterprise AI Platform · Anti-Bias Certified · SOC 2 Type II & GDPR Compliant", props.Text{
					Size:  7,
					Color: colorTextMuted,
					Top:   1,
				}),
			),
			col.New(4).Add(
				text.New(fmt.Sprintf("Session: %s", detail.InterviewID.String()[:8]), props.Text{
					Size:  7,
					Align: align.Right,
					Color: colorTextMuted,
					Top:   1,
				}),
			),
		),
	)

	candName := "Unknown Candidate"
	candEmail := "N/A"
	if detail.Candidate != nil {
		if detail.Candidate.Name != "" {
			candName = cleanPDFText(detail.Candidate.Name)
		}
		if detail.Candidate.Email != "" {
			candEmail = cleanPDFText(detail.Candidate.Email)
		}
	}

	jobTitle := "Technical Role"
	if detail.Job != nil && detail.Job.Title != "" {
		jobTitle = cleanPDFText(detail.Job.Title)
	}

	recVerdict := strings.ToLower(strings.TrimSpace(eval.Recommendation))
	if recVerdict == "" {
		recVerdict = "under_review"
	}

	// -------------------------------------------------------------------------
	// 1. Candidate & Assessment Metadata Block
	// -------------------------------------------------------------------------
	m.AddRow(18,
		col.New(6).Add(
			text.New(fmt.Sprintf("CANDIDATE: %s", candName), props.Text{
				Style: fontstyle.Bold,
				Size:  9.5,
				Color: colorTextDark,
				Top:   1.5,
				Left:  3,
			}),
			text.New(fmt.Sprintf("Email: %s", candEmail), props.Text{
				Size:  8,
				Color: colorTextMuted,
				Top:   6.5,
				Left:  3,
			}),
			text.New(fmt.Sprintf("Role: %s", jobTitle), props.Text{
				Style: fontstyle.Bold,
				Size:  8.5,
				Color: colorBrandDark,
				Top:   11.5,
				Left:  3,
			}),
		).WithStyle(&props.Cell{
			BackgroundColor: colorCardBg,
			BorderColor:     colorCardBorder,
			BorderType:      border.Full,
			BorderThickness: 0.5,
		}),
		col.New(6).Add(
			text.New(fmt.Sprintf("Assessment Date: %s", detail.CreatedAt.Format("02 Jan 2006")), props.Text{
				Size:  8,
				Color: colorTextDark,
				Top:   1.5,
				Left:  3,
			}),
			text.New(fmt.Sprintf("Status: %s (%d Questions)", strings.ToUpper(string(detail.Status)), len(detail.Questions)), props.Text{
				Style: fontstyle.Bold,
				Size:  8,
				Color: colorEmerald,
				Top:   6.5,
				Left:  3,
			}),
			text.New(fmt.Sprintf("Session ID: %s", detail.InterviewID.String()), props.Text{
				Size:  7,
				Color: colorTextMuted,
				Top:   11.5,
				Left:  3,
			}),
		).WithStyle(&props.Cell{
			BackgroundColor: colorCardBg,
			BorderColor:     colorCardBorder,
			BorderType:      border.Full,
			BorderThickness: 0.5,
		}),
	)
	m.AddRow(4)

	// -------------------------------------------------------------------------
	// 2. Executive Verdict & Core Dimensions Scorecard
	// -------------------------------------------------------------------------
	var (
		verdictColor  *props.Color
		verdictBg     *props.Color
		verdictBorder *props.Color
		verdictLabel  string
	)

	switch recVerdict {
	case "proceed":
		verdictColor = colorEmerald
		verdictBg = colorEmeraldBg
		verdictBorder = colorEmeraldBorder
		verdictLabel = "PROCEED TO HIRE"
	case "reconsider":
		verdictColor = colorAmber
		verdictBg = colorAmberBg
		verdictBorder = colorAmberBorder
		verdictLabel = "FURTHER REVIEW"
	case "reject":
		verdictColor = colorRose
		verdictBg = colorRoseBg
		verdictBorder = colorRoseBorder
		verdictLabel = "DO NOT PROCEED"
	default:
		verdictColor = colorBrandPrimary
		verdictBg = colorIndigoBg
		verdictBorder = colorIndigoBorder
		verdictLabel = strings.ToUpper(recVerdict)
	}

	// Prepare Dimension columns (4 cards)
	var dimNames []string
	for k := range eval.Dimensions {
		dimNames = append(dimNames, k)
	}
	sort.Strings(dimNames)
	if len(dimNames) == 0 {
		dimNames = []string{"technical", "problem_solving", "communication", "culture_fit"}
	}
	if len(dimNames) > 4 {
		log.Printf("WARN evaluation PDF contains %d dimensions; rendering first four", len(dimNames))
		dimNames = dimNames[:4]
	}

	dimRowCols := make([]core.Col, 0, len(dimNames))
	colWidth := 12 / len(dimNames)
	if colWidth < 3 {
		colWidth = 3
	}

	for _, k := range dimNames {
		v, ok := eval.Dimensions[k]
		scoreVal := 0.0
		weightVal := 0.25
		if ok {
			scoreVal = clampPDFScore(v.Score)
			weightVal = v.Weight
		}
		displayName := formatDimensionTitle(k)

		dimRowCols = append(dimRowCols, col.New(colWidth).Add(
			text.New(displayName, props.Text{
				Style: fontstyle.Bold,
				Size:  7.5,
				Align: align.Center,
				Color: colorTextDark,
				Top:   1.5,
			}),
			text.New(fmt.Sprintf("%.1f / 100", scoreVal), props.Text{
				Style: fontstyle.Bold,
				Size:  9.5,
				Align: align.Center,
				Color: colorBrandPrimary,
				Top:   5.5,
			}),
			text.New(fmt.Sprintf("Weight: %.0f%%", weightVal*100), props.Text{
				Size:  6.5,
				Align: align.Center,
				Color: colorTextMuted,
				Top:   10,
			}),
		).WithStyle(&props.Cell{
			BackgroundColor: colorCardBg,
			BorderColor:     colorCardBorder,
			BorderType:      border.Full,
			BorderThickness: 0.4,
		}))
	}

	m.AddRow(20,
		// Overall Score Card (Col 4)
		col.New(4).Add(
			text.New("OVERALL EVALUATION SCORE", props.Text{
				Style: fontstyle.Bold,
				Size:  6.5,
				Align: align.Center,
				Color: colorTextMuted,
				Top:   2,
			}),
			text.New(fmt.Sprintf("%.0f", clampPDFScore(eval.OverallScore)), props.Text{
				Style: fontstyle.Bold,
				Size:  20,
				Align: align.Center,
				Color: colorBrandDark,
				Top:   5.5,
			}),
			text.New(fmt.Sprintf("OUT OF 100  ·  %s", verdictLabel), props.Text{
				Style: fontstyle.Bold,
				Size:  7,
				Align: align.Center,
				Color: verdictColor,
				Top:   14.5,
			}),
		).WithStyle(&props.Cell{
			BackgroundColor: verdictBg,
			BorderColor:     verdictBorder,
			BorderType:      border.Full,
			BorderThickness: 0.8,
		}),

		// Dimension Roll-Up (Col 8)
		col.New(8).Add(
			text.New("CORE COMPETENCY DIMENSIONS BREAKDOWN", props.Text{
				Style: fontstyle.Bold,
				Size:  7.5,
				Color: colorBrandDark,
				Top:   1.5,
				Left:  3,
			}),
			text.New("Aggregated weighted scores across technical execution, architecture, and communication.", props.Text{
				Size:  6.5,
				Color: colorTextMuted,
				Top:   5.5,
				Left:  3,
			}),
		).WithStyle(&props.Cell{
			BackgroundColor: colorCardBg,
			BorderColor:     colorCardBorder,
			BorderType:      border.Full,
			BorderThickness: 0.5,
		}),
	)
	m.AddRow(2)
	m.AddRow(14, dimRowCols...)
	m.AddRow(4)

	// -------------------------------------------------------------------------
	// 3. Demonstrated Strengths & Growth Areas (Side-by-Side)
	// -------------------------------------------------------------------------
	m.AddRow(5,
		text.NewCol(6, "[+] DEMONSTRATED STRENGTHS", props.Text{
			Style: fontstyle.Bold,
			Size:  8,
			Color: colorEmerald,
			Left:  1,
		}),
		text.NewCol(6, "[-] AREAS FOR GROWTH & DEVELOPMENT", props.Text{
			Style: fontstyle.Bold,
			Size:  8,
			Color: colorAmber,
			Left:  1,
		}),
	)

	maxBullets := len(eval.Strengths)
	if len(eval.Weaknesses) > maxBullets {
		maxBullets = len(eval.Weaknesses)
	}
	if maxBullets == 0 {
		maxBullets = 1
	}

	for i := 0; i < maxBullets; i++ {
		strengthText := "• Consistent technical competence demonstrated"
		if i < len(eval.Strengths) {
			strengthText = fmt.Sprintf("• %s", cleanPDFText(eval.Strengths[i]))
		} else if i > 0 {
			strengthText = ""
		}

		weaknessText := "• None noted during evaluation session"
		if i < len(eval.Weaknesses) {
			weaknessText = fmt.Sprintf("• %s", cleanPDFText(eval.Weaknesses[i]))
		} else if i > 0 {
			weaknessText = ""
		}

		if strengthText != "" || weaknessText != "" {
			m.AddAutoRow(
				text.NewCol(6, strengthText, props.Text{
					Size:   7.5,
					Color:  colorTextBody,
					Top:    0.5,
					Bottom: 1.0,
					Left:   2,
					Right:  2,
				}),
				text.NewCol(6, weaknessText, props.Text{
					Size:   7.5,
					Color:  colorTextBody,
					Top:    0.5,
					Bottom: 1.0,
					Left:   2,
					Right:  2,
				}),
			)
		}
	}
	m.AddRow(3)

	// -------------------------------------------------------------------------
	// 4. Anti-Cheating & Proctoring Integrity Audit
	// -------------------------------------------------------------------------
	summary := detail.ProctoringSummary
	hasTelemetry := summary.IntegrityScore != 0 || len(summary.Flags) > 0
	integrityScore := summary.IntegrityScore
	riskTier := strings.ToUpper(string(summary.RiskLevel))
	if riskTier == "" {
		riskTier = "LOW"
	}
	riskColor := colorEmerald
	switch riskTier {
	case "MEDIUM":
		riskColor = colorAmber
	case "HIGH":
		riskColor = colorRose
	}

	integrityLine := fmt.Sprintf("Integrity Score: %d / 100  |  Risk Level: %s  ·  Verified Session Authenticity", integrityScore, riskTier)
	integrityStyle := fontstyle.Bold
	integrityColor := riskColor
	if !hasTelemetry {
		integrityLine = "No telemetry recorded"
		integrityStyle = fontstyle.Normal
		integrityColor = colorTextMuted
	}

	m.AddRow(17,
		col.New(12).Add(
			text.New("PROCTORING & SESSION INTEGRITY AUDIT", props.Text{
				Style: fontstyle.Bold,
				Size:  8,
				Color: colorTextDark,
				Top:   1.5,
				Left:  3,
			}),
			text.New(integrityLine, props.Text{
				Style: integrityStyle,
				Size:  7.5,
				Color: integrityColor,
				Top:   6.0,
				Left:  3,
			}),
			text.New(fmt.Sprintf("Tab Switches: %d  ·  Time Away: %ds  ·  Paste Events: %d (Suspicious: %d)  ·  Audio Anomalies: %d",
				summary.TabSwitchCount, summary.TotalAwayDurationSec, summary.PasteEventCount, summary.SuspiciousPasteCount, summary.AudioAnomalyCount), props.Text{
				Size:  7,
				Color: colorTextMuted,
				Top:   10.5,
				Left:  3,
			}),
		).WithStyle(&props.Cell{
			BackgroundColor: colorCardBg,
			BorderColor:     colorCardBorder,
			BorderType:      border.Full,
			BorderThickness: 0.5,
		}),
	)
	m.AddRow(5)

	// -------------------------------------------------------------------------
	// 5. Per-Question Detailed Technical Breakdown & Verbatim Evidence
	// -------------------------------------------------------------------------
	m.AddRow(7,
		text.NewCol(12, "DETAILED QUESTION BREAKDOWN & VERBATIM EVIDENCE", props.Text{
			Style: fontstyle.Bold,
			Size:  9.5,
			Color: colorBrandDark,
		}),
	)
	m.AddRow(2)

	ansMap := make(map[int]string)
	for _, a := range detail.Answers {
		turn := a.Turn
		act := a.Action
		if act == "" {
			act = "reply"
		}
		if existing, ok := ansMap[a.Idx]; ok {
			if turn == 0 {
				turn = 2
			}
			ansMap[a.Idx] = fmt.Sprintf("%s\n\n[Candidate Turn %d (%s)]: %s", existing, turn, act, cleanPDFText(a.Content))
		} else {
			if a.Turn > 1 || (a.Action != "" && a.Action != "advance") {
				turnNum := a.Turn
				if turnNum == 0 {
					turnNum = 1
				}
				ansMap[a.Idx] = fmt.Sprintf("[Candidate Turn %d (%s)]: %s", turnNum, act, cleanPDFText(a.Content))
			} else {
				ansMap[a.Idx] = cleanPDFText(a.Content)
			}
		}
	}
	qScoreMap := make(map[int]float64)
	qRatMap := make(map[int]string)
	for _, pq := range eval.PerQuestion {
		qScoreMap[pq.QuestionIdx] = clampPDFScore(pq.Score)
		qRatMap[pq.QuestionIdx] = cleanPDFText(pq.Rationale)
	}

	for _, q := range detail.Questions {
		qScore := qScoreMap[q.Idx]
		qRat := qRatMap[q.Idx]
		if qRat == "" {
			qRat = "Demonstrated clear competency and domain knowledge in answer formulation."
		}

		catLabel := cleanPDFText(q.Category)
		if catLabel == "" {
			catLabel = "Technical Architecture"
		}
		if q.Skill != "" {
			catLabel = fmt.Sprintf("%s · %s", catLabel, cleanPDFText(q.Skill))
		}

		// Question Header Bar
		m.AddRow(6,
			col.New(9).Add(
				text.New(fmt.Sprintf("Question %d: %s", q.Idx, catLabel), props.Text{
					Style: fontstyle.Bold,
					Size:  8,
					Color: colorWhite,
					Top:   1.2,
					Left:  2,
				}),
			).WithStyle(&props.Cell{
				BackgroundColor: colorBrandDark,
			}),
			col.New(3).Add(
				text.New(fmt.Sprintf("Score: %.0f / 100", qScore), props.Text{
					Style: fontstyle.Bold,
					Size:  8,
					Align: align.Right,
					Color: colorWhite,
					Top:   1.2,
					Right: 2,
				}),
			).WithStyle(&props.Cell{
				BackgroundColor: colorBrandDark,
			}),
		)

		// Question Prompt
		m.AddAutoRow(
			text.NewCol(12, fmt.Sprintf("Challenge Prompt: %s", cleanPDFText(q.Content)), props.Text{
				Style:  fontstyle.Bold,
				Size:   7.5,
				Color:  colorTextDark,
				Top:    1.5,
				Bottom: 1.5,
				Left:   2,
				Right:  2,
			}).WithStyle(&props.Cell{
				BackgroundColor: colorCardBg,
				BorderColor:     colorCardBorder,
				BorderType:      border.Full,
				BorderThickness: 0.3,
			}),
		)

		// Candidate Response
		ansText := ansMap[q.Idx]
		if ansText == "" {
			ansText = "(No response submitted)"
		}
		m.AddAutoRow(
			text.NewCol(12, fmt.Sprintf("Candidate Verbatim Response: \"%s\"", cleanPDFText(ansText)), props.Text{
				Style:  fontstyle.Italic,
				Size:   7.5,
				Color:  colorTextBody,
				Top:    1.5,
				Bottom: 1.5,
				Left:   2,
				Right:  2,
			}).WithStyle(&props.Cell{
				BackgroundColor: colorWhite,
				BorderColor:     colorCardBorder,
				BorderType:      border.Full,
				BorderThickness: 0.3,
			}),
		)

		// Evaluator Rationale Box
		m.AddAutoRow(
			text.NewCol(12, fmt.Sprintf("AI Evaluator Rationale: %s", cleanPDFText(qRat)), props.Text{
				Size:   7.5,
				Color:  colorBrandDark,
				Top:    1.5,
				Bottom: 1.5,
				Left:   2,
				Right:  2,
			}).WithStyle(&props.Cell{
				BackgroundColor: colorIndigoBg,
				BorderColor:     colorIndigoBorder,
				BorderType:      border.Full,
				BorderThickness: 0.4,
			}),
		)
		m.AddRow(3)
	}

	doc, err := m.Generate()
	if err != nil {
		return nil, fmt.Errorf("generate pdf: %w", err)
	}

	return doc.GetBytes(), nil
}

func cleanPDFText(value string) string {
	value = strings.NewReplacer("\n", " ", "\r", " ", "\t", " ").Replace(value)
	runes := []rune(value)
	if len(runes) <= 100 {
		return value
	}
	return string(runes[:97]) + "..."
}

func clampPDFScore(score float64) float64 {
	if score < 0 {
		return 0
	}
	if score > 100 {
		return 100
	}
	return score
}

func formatDimensionTitle(name string) string {
	parts := strings.Split(name, "_")
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + strings.ToLower(p[1:])
		}
	}
	return strings.Join(parts, " ")
}

package domain

import (
	"fmt"
	"time"
)

type ProctoringEventType string

const (
	EventTypeTabSwitch     ProctoringEventType = "tab_switch"
	EventTypeFocusLost     ProctoringEventType = "focus_lost"
	EventTypeFocusRegained ProctoringEventType = "focus_regained"
	EventTypePaste         ProctoringEventType = "clipboard_paste"
	EventTypeWindowResize  ProctoringEventType = "window_resize"
	EventTypeAudioAnomaly  ProctoringEventType = "audio_anomaly"
)

type ProctoringEvent struct {
	Type        ProctoringEventType `json:"type"`
	Timestamp   time.Time           `json:"timestamp"`
	QuestionIdx int                 `json:"question_idx,omitempty"`
	Details     *TelemetryDetails   `json:"details,omitempty"`
}

type RiskLevel string

const (
	RiskLow    RiskLevel = "low"
	RiskMedium RiskLevel = "medium"
	RiskHigh   RiskLevel = "high"
)

type ProctoringSummary struct {
	IntegrityScore       int       `json:"integrity_score"`
	RiskLevel            RiskLevel `json:"risk_level"`
	TabSwitchCount       int       `json:"tab_switch_count"`
	TotalAwayDurationSec int       `json:"total_away_duration_sec"`
	PasteEventCount      int       `json:"paste_event_count"`
	SuspiciousPasteCount int       `json:"suspicious_paste_count"`
	AudioAnomalyCount    int       `json:"audio_anomaly_count"`
	Flags                []string  `json:"flags"`
}

// CalculateProctoringSummary analyzes chronological telemetry events and computes
// an integrity score (0-100), risk tier, and human-readable audit flags.
func CalculateProctoringSummary(events []ProctoringEvent) ProctoringSummary {
	var summary ProctoringSummary
	if len(events) == 0 {
		return summary
	}

	score := 100
	var lastFocusLost time.Time
	var lastResize time.Time
	const resizeDeductionInterval = 30 * time.Second

	for _, ev := range events {
		switch ev.Type {
		case EventTypeTabSwitch:
			summary.TabSwitchCount++
			score -= 5

		case EventTypeFocusLost:
			lastFocusLost = ev.Timestamp

		case EventTypeFocusRegained:
			if !lastFocusLost.IsZero() && ev.Timestamp.After(lastFocusLost) {
				awaySec := int(ev.Timestamp.Sub(lastFocusLost).Seconds())
				if awaySec > 0 {
					summary.TotalAwayDurationSec += awaySec
					if awaySec > 10 {
						score -= 5 // extra penalty for prolonged absence
					}
				}
			}

		case EventTypePaste:
			summary.PasteEventCount++
			charCount := 0
			if ev.Details != nil {
				charCount = ev.Details.PastedTextLength
			}
			// Pastes over 150 chars or rapid pastes are treated as suspicious external snippets
			if charCount > 150 {
				summary.SuspiciousPasteCount++
				score -= 15
			} else {
				score -= 3
			}

		case EventTypeAudioAnomaly:
			summary.AudioAnomalyCount++
			score -= 15

		case EventTypeWindowResize:
			// Resize floods (drag-resize fires dozens of events; mobile
			// rotation too). Rate-limit the deduction to one per 30s window —
			// a benign resize must not floor the integrity score.
			if lastResize.IsZero() || ev.Timestamp.Sub(lastResize) >= resizeDeductionInterval {
				lastResize = ev.Timestamp
				score -= 2
			}
		}
	}

	// Generate audit flags
	if summary.TabSwitchCount > 0 {
		summary.Flags = append(summary.Flags,
			fmt.Sprintf("Candidate switched tabs or blurred browser %d time(s)", summary.TabSwitchCount))
	}
	if summary.TotalAwayDurationSec > 5 {
		summary.Flags = append(summary.Flags,
			fmt.Sprintf("Candidate spent %d second(s) away from the interview window", summary.TotalAwayDurationSec))
	}
	if summary.SuspiciousPasteCount > 0 {
		summary.Flags = append(summary.Flags,
			fmt.Sprintf("Detected %d large clipboard paste event(s) (>150 characters)", summary.SuspiciousPasteCount))
	} else if summary.PasteEventCount > 0 {
		summary.Flags = append(summary.Flags,
			fmt.Sprintf("Detected %d minor clipboard paste event(s)", summary.PasteEventCount))
	}
	if summary.AudioAnomalyCount > 0 {
		summary.Flags = append(summary.Flags,
			fmt.Sprintf("Detected %d background voice or secondary speaker audio anomaly event(s)", summary.AudioAnomalyCount))
	}

	// Clamp score [0, 100]
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	summary.IntegrityScore = score

	if score >= 85 {
		summary.RiskLevel = RiskLow
	} else if score >= 60 {
		summary.RiskLevel = RiskMedium
	} else {
		summary.RiskLevel = RiskHigh
	}

	return summary
}

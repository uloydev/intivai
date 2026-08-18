package domain_test

import (
	"testing"
	"time"

	"github.com/intivai/backend/internal/interview/domain"
	"github.com/stretchr/testify/assert"
)

func TestDefaultProctoringSummary(t *testing.T) {
	summary := domain.DefaultProctoringSummary()
	assert.Equal(t, 100, summary.IntegrityScore)
	assert.Equal(t, domain.RiskLow, summary.RiskLevel)
	assert.Equal(t, 0, summary.TabSwitchCount)
	assert.Empty(t, summary.Flags)
}

func TestCalculateProctoringSummary_CleanSession(t *testing.T) {
	summary := domain.CalculateProctoringSummary([]domain.ProctoringEvent{})
	assert.Equal(t, 100, summary.IntegrityScore)
	assert.Equal(t, domain.RiskLow, summary.RiskLevel)
	assert.Empty(t, summary.Flags)
}

func TestCalculateProctoringSummary_TabSwitchesAndAbsence(t *testing.T) {
	now := time.Now()
	events := []domain.ProctoringEvent{
		{
			Type:        domain.EventTypeTabSwitch,
			Timestamp:   now,
			QuestionIdx: 1,
		},
		{
			Type:        domain.EventTypeFocusLost,
			Timestamp:   now.Add(1 * time.Minute),
			QuestionIdx: 2,
		},
		{
			Type:        domain.EventTypeFocusRegained,
			Timestamp:   now.Add(1*time.Minute + 15*time.Second),
			QuestionIdx: 2,
		},
	}

	summary := domain.CalculateProctoringSummary(events)
	assert.Equal(t, 1, summary.TabSwitchCount)
	assert.Equal(t, 15, summary.TotalAwayDurationSec)
	// 100 - 5 (tab switch) - 5 (away > 10s) = 90
	assert.Equal(t, 90, summary.IntegrityScore)
	assert.Equal(t, domain.RiskLow, summary.RiskLevel)
	assert.Len(t, summary.Flags, 2)
}

func TestCalculateProctoringSummary_SuspiciousPastesAndAudio(t *testing.T) {
	now := time.Now()
	events := []domain.ProctoringEvent{
		{
			Type:        domain.EventTypePaste,
			Timestamp:   now,
			QuestionIdx: 1,
			Details:     map[string]interface{}{"char_count": 350},
		},
		{
			Type:        domain.EventTypeAudioAnomaly,
			Timestamp:   now.Add(30 * time.Second),
			QuestionIdx: 2,
			Details:     map[string]interface{}{"anomaly": "multiple_speakers"},
		},
		{
			Type:        domain.EventTypeTabSwitch,
			Timestamp:   now.Add(45 * time.Second),
			QuestionIdx: 2,
		},
		{
			Type:        domain.EventTypeTabSwitch,
			Timestamp:   now.Add(50 * time.Second),
			QuestionIdx: 2,
		},
		{
			Type:        domain.EventTypeTabSwitch,
			Timestamp:   now.Add(55 * time.Second),
			QuestionIdx: 2,
		},
	}

	summary := domain.CalculateProctoringSummary(events)
	assert.Equal(t, 1, summary.SuspiciousPasteCount)
	assert.Equal(t, 1, summary.AudioAnomalyCount)
	assert.Equal(t, 3, summary.TabSwitchCount)
	// 100 - 15 (large paste) - 15 (audio anomaly) - 15 (3 tab switches) = 55
	assert.Equal(t, 55, summary.IntegrityScore)
	assert.Equal(t, domain.RiskHigh, summary.RiskLevel)
	assert.NotEmpty(t, summary.Flags)
}

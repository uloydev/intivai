package service

import (
	"strings"
	"testing"
)

func TestDetectBias(t *testing.T) {
	biased := []string{
		"How old are you?",
		"Are you married?",
		"What religion do you follow?",
		"Which party do you vote for?",
		"Do you plan to have children?",
		"Are you a man or a woman?",
	}
	for _, q := range biased {
		if !IsBiased(q) {
			t.Errorf("bias not detected: %q", q)
		}
	}
}

func TestDetectBiasClean(t *testing.T) {
	clean := []string{
		"Describe a project where you used Go concurrency.",
		"How do you handle tight deadlines?",
		"Walk me through your experience with PostgreSQL indexing.",
		"Tell me about a time you disagreed with a teammate.",
	}
	for _, q := range clean {
		if IsBiased(q) {
			t.Errorf("false positive: %q -> %v", q, DetectBias(q))
		}
	}
}

func TestDetectBiasCategory(t *testing.T) {
	flagged := DetectBias("Are you married or single?")
	if len(flagged) != 1 || flagged[0] != "marital_family" {
		t.Fatalf("flagged = %v", flagged)
	}
}

func TestDetectBiasMultipleCategories(t *testing.T) {
	flagged := DetectBias("How old are you and what religion do you follow?")
	if len(flagged) != 2 {
		t.Fatalf("flagged = %v", flagged)
	}
	joined := strings.Join(flagged, ",")
	if !strings.Contains(joined, "age") || !strings.Contains(joined, "religion") {
		t.Fatalf("flagged = %v", flagged)
	}
}

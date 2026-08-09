package service

import (
	"strings"
	"testing"
)

func TestShouldProbeShortAnswer(t *testing.T) {
	if !ShouldProbe(ProbeInput{Answer: "Yes."}) {
		t.Fatal("one-word answer must probe")
	}
	if !ShouldProbe(ProbeInput{Answer: strings.Repeat("word ", 7)}) {
		t.Fatal("7-word answer must probe")
	}
}

func TestShouldProbeDetailedAnswer(t *testing.T) {
	if ShouldProbe(ProbeInput{Answer: strings.Repeat("word ", 25)}) {
		t.Fatal("25-word answer must not probe")
	}
	if ShouldProbe(ProbeInput{Answer: "I built the payment service with Go, Postgres and Kafka, handling 10k rps in production."}) {
		t.Fatal("detailed answer must not probe")
	}
}

func TestProbeQuestionMentionsSkillAndCategory(t *testing.T) {
	q := ProbeQuestion("technical", "kubernetes")
	if q.Prompt == "" || !strings.Contains(q.Prompt, "kubernetes") {
		t.Fatalf("probe must reference the skill: %q", q.Prompt)
	}
	if !strings.Contains(q.Prompt, "example") {
		t.Fatalf("probe must ask for a concrete example: %q", q.Prompt)
	}
	if q.Category != "technical" {
		t.Fatalf("category = %q, want technical", q.Category)
	}
}

func TestProbeQuestionFallsBackToCategory(t *testing.T) {
	q := ProbeQuestion("behavioral", "")
	if !strings.Contains(q.Prompt, "behavioral") {
		t.Fatalf("probe without skill must reference category: %q", q.Prompt)
	}
}

package service

import (
	"strings"
)

// Dynamic follow-up probing (Research §2: "If candidate shows weakness →
// probe deeper; strength → move to next topic"). Deterministic, no LLM cost:
// a shallow answer (few words) on a skill question triggers a probe follow-up
// asking for a concrete example; detailed answers advance to the next topic.

// MinAnswerWords — answers shorter than this are treated as shallow.
const MinAnswerWords = 8

type ProbeInput struct {
	Answer           string
	QuestionCategory string
	QuestionSkill    string
}

// ShouldProbe reports whether the answer was too shallow to assess.
func ShouldProbe(in ProbeInput) bool {
	words := strings.Fields(strings.TrimSpace(in.Answer))
	return len(words) < MinAnswerWords
}

// ProbeQuestion builds the follow-up question for the current topic.
func ProbeQuestion(category, skill string) Question {
	topic := skill
	if topic == "" {
		topic = category
	}
	return Question{
		Prompt:   "Your answer was brief. Could you elaborate on " + topic + " and give a concrete example from your experience?",
		Category: category,
		Skill:    skill,
	}
}

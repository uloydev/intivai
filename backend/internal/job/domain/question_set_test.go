package domain

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func validQuestions() []Question {
	return []Question{
		{
			Category:    QuestionCategoryTechnical,
			Prompt:      "Walk me through how you designed a Go service that had to survive a partial outage.",
			Context:     "The role owns production Go services, so resilience decisions are part of the daily work.",
			Expectation: "A concrete architecture, the failure modes considered, and the tradeoffs accepted.",
			Skill:       "go",
			Priority:    1,
		},
		{
			Category:    QuestionCategoryBehavioral,
			Prompt:      "Tell me about a time a teammate disagreed with your technical decision.",
			Context:     "This team reviews every design in the open, so disagreement is routine here.",
			Expectation: "Evidence the candidate listens, argues with data, and commits once a call is made.",
			Priority:    3,
		},
	}
}

// The descriptive fields are the whole point of D5: a stored question must
// round-trip its prompt AND its framing, otherwise the candidate sees a bare
// question and the recruiter loses the rationale.
func TestQuestionSetMarshalRoundTripKeepsDescriptiveFields(t *testing.T) {
	at := time.Date(2026, 8, 24, 10, 0, 0, 0, time.UTC)
	set := NewQuestionSet(QuestionSourceLLM, validQuestions(), at)

	raw, err := set.Marshal()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	// Wire keys are part of the stored contract — renaming them silently
	// orphans every previously generated set.
	for _, key := range []string{`"version"`, `"source"`, `"generated_at"`, `"questions"`, `"prompt"`, `"context"`, `"expectation"`} {
		if !strings.Contains(string(raw), key) {
			t.Fatalf("stored payload missing %s: %s", key, raw)
		}
	}

	back, err := ParseQuestionSet(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if back.Version != QuestionSetVersion {
		t.Fatalf("version = %d, want %d", back.Version, QuestionSetVersion)
	}
	if back.Source != QuestionSourceLLM {
		t.Fatalf("source = %q, want %q", back.Source, QuestionSourceLLM)
	}
	if !back.GeneratedAt.Equal(at) {
		t.Fatalf("generated_at = %v, want %v", back.GeneratedAt, at)
	}
	if len(back.Questions) != 2 {
		t.Fatalf("questions = %d, want 2", len(back.Questions))
	}
	want := validQuestions()[0]
	got := back.Questions[0]
	if got.Prompt != want.Prompt || got.Context != want.Context || got.Expectation != want.Expectation {
		t.Fatalf("descriptive fields lost: %+v", got)
	}
	if got.Skill != want.Skill || got.Priority != want.Priority || got.Category != want.Category {
		t.Fatalf("question metadata lost: %+v", got)
	}
}

// Strict validation is what makes "malformed LLM JSON -> retry, never partial
// persistence" enforceable: the worker marshals through Validate.
func TestQuestionSetValidateRejectsIncompleteQuestions(t *testing.T) {
	cases := []struct {
		name string
		set  *QuestionSet
	}{
		{"no questions", &QuestionSet{Version: QuestionSetVersion, Source: QuestionSourceLLM}},
		{"bad version", &QuestionSet{Version: 0, Source: QuestionSourceLLM, Questions: validQuestions()}},
		{"unknown source", &QuestionSet{Version: QuestionSetVersion, Source: "guesswork", Questions: validQuestions()}},
		{"missing prompt", &QuestionSet{Version: QuestionSetVersion, Source: QuestionSourceLLM, Questions: []Question{
			{Category: QuestionCategoryTechnical, Context: "c", Expectation: "e"},
		}}},
		{"missing context", &QuestionSet{Version: QuestionSetVersion, Source: QuestionSourceLLM, Questions: []Question{
			{Category: QuestionCategoryTechnical, Prompt: "p", Expectation: "e"},
		}}},
		{"missing expectation", &QuestionSet{Version: QuestionSetVersion, Source: QuestionSourceLLM, Questions: []Question{
			{Category: QuestionCategoryTechnical, Prompt: "p", Context: "c"},
		}}},
		{"unknown category", &QuestionSet{Version: QuestionSetVersion, Source: QuestionSourceLLM, Questions: []Question{
			{Category: "trivia", Prompt: "p", Context: "c", Expectation: "e"},
		}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.set.Validate(); err == nil {
				t.Fatal("expected validation error")
			}
			if _, err := tc.set.Marshal(); err == nil {
				t.Fatal("Marshal must refuse an invalid set (no partial persistence)")
			}
		})
	}
}

// Fallback selection: anything the interview cannot trust must resolve to
// "use the deterministic templates" rather than half-render a broken set.
func TestSelectQuestionSetFallbackSelection(t *testing.T) {
	good, err := NewQuestionSet(QuestionSourceLLM, validQuestions(), time.Now().UTC()).Marshal()
	if err != nil {
		t.Fatal(err)
	}
	futureVersion, err := json.Marshal(map[string]any{
		"version":   QuestionSetVersion + 1,
		"source":    QuestionSourceLLM,
		"questions": validQuestions(),
	})
	if err != nil {
		t.Fatal(err)
	}
	incomplete, err := json.Marshal(map[string]any{
		"version": QuestionSetVersion,
		"source":  QuestionSourceLLM,
		"questions": []map[string]any{
			{"category": "technical", "prompt": "no framing at all"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name    string
		raw     json.RawMessage
		wantOK  bool
		wantLen int
	}{
		{"nil column", nil, false, 0},
		{"empty column", json.RawMessage(``), false, 0},
		{"sql null", json.RawMessage(`null`), false, 0},
		{"malformed json", json.RawMessage(`{"questions":[`), false, 0},
		{"empty questions", json.RawMessage(`{"version":1,"source":"llm","questions":[]}`), false, 0},
		{"unknown future version", json.RawMessage(futureVersion), false, 0},
		{"questions without framing", json.RawMessage(incomplete), false, 0},
		{"usable set", good, true, 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			set, ok := SelectQuestionSet(tc.raw)
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tc.wantOK)
			}
			if !tc.wantOK {
				if set != nil {
					t.Fatalf("unusable set must be nil, got %+v", set)
				}
				return
			}
			if len(set.Questions) != tc.wantLen {
				t.Fatalf("questions = %d, want %d", len(set.Questions), tc.wantLen)
			}
			if !set.Usable() {
				t.Fatal("selected set reports itself unusable")
			}
		})
	}
}

// Template-sourced sets are legal too (offline fallback may be persisted by a
// later batch); the version/source contract must accept them.
func TestQuestionSetAcceptsTemplateSource(t *testing.T) {
	set := NewQuestionSet(QuestionSourceTemplate, validQuestions(), time.Now().UTC())
	if err := set.Validate(); err != nil {
		t.Fatalf("template source rejected: %v", err)
	}
	if set.Version != QuestionSetVersion {
		t.Fatalf("version = %d, want %d", set.Version, QuestionSetVersion)
	}
}

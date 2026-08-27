package service

import (
	"testing"
	"time"

	jobdomain "github.com/intivai/backend/internal/job/domain"
)

func storedSet(qs ...jobdomain.Question) *jobdomain.QuestionSet {
	return jobdomain.NewQuestionSet(jobdomain.QuestionSourceLLM, qs, time.Date(2026, 8, 24, 9, 0, 0, 0, time.UTC))
}

func llmQuestions() []jobdomain.Question {
	return []jobdomain.Question{
		{
			Category:    jobdomain.QuestionCategoryTechnical,
			Prompt:      "Describe the trickiest data migration you shipped and how you verified it.",
			Context:     "This role owns schema changes on a live product, so migration discipline matters.",
			Expectation: "A rollout plan, verification steps, and what the candidate would do differently.",
			Skill:       "postgresql",
			Priority:    1,
		},
		{
			Category:    jobdomain.QuestionCategorySituational,
			Prompt:      "A release is half deployed when errors spike. What do you do first?",
			Context:     "On-call is shared by the whole team, so incident instincts are part of the job.",
			Expectation: "Stop the bleeding first, communicate, then investigate root cause.",
			Priority:    2,
		},
	}
}

// D5 core invariant: when a job carries a stored LLM set, the interview uses
// it VERBATIM. No re-templating, no per-candidate variation.
func TestStoredSetUsedVerbatim(t *testing.T) {
	set := storedSet(llmQuestions()...)
	cv := CandidateProfile{Skills: []string{"Go"}}
	job := JobRequirements{Title: "Backend Engineer", RequiredSkills: []string{"Go", "Kubernetes"}}

	qs := GenerateQuestions(cv, job, 5, set)
	if len(qs) != 2 {
		t.Fatalf("questions = %d, want the 2 stored ones", len(qs))
	}
	for i, want := range llmQuestions() {
		if qs[i].Prompt != want.Prompt {
			t.Fatalf("question %d prompt = %q, want %q", i, qs[i].Prompt, want.Prompt)
		}
		if qs[i].Context != want.Context || qs[i].Expectation != want.Expectation {
			t.Fatalf("question %d lost its framing: %+v", i, qs[i])
		}
		if qs[i].Category != want.Category || qs[i].Skill != want.Skill {
			t.Fatalf("question %d metadata mismatch: %+v", i, qs[i])
		}
	}
}

// The stored set is generated ONCE PER JOB at publish, never per candidate:
// two very different CVs must hear exactly the same questions.
func TestStoredSetIsIdenticalAcrossCandidates(t *testing.T) {
	set := storedSet(llmQuestions()...)
	job := JobRequirements{Title: "Backend Engineer", RequiredSkills: []string{"Go", "Kubernetes"}}

	senior := GenerateQuestions(CandidateProfile{Skills: []string{"Go", "Kubernetes", "PostgreSQL"}, ExperienceYears: 9}, job, 5, set)
	junior := GenerateQuestions(CandidateProfile{Skills: []string{"Python"}, ExperienceYears: 0}, job, 5, set)

	if len(senior) != len(junior) {
		t.Fatalf("per-candidate divergence: %d vs %d questions", len(senior), len(junior))
	}
	for i := range senior {
		if senior[i] != junior[i] {
			t.Fatalf("question %d differs per candidate: %+v vs %+v", i, senior[i], junior[i])
		}
	}
}

func TestStoredSetRespectsLimit(t *testing.T) {
	set := storedSet(llmQuestions()...)
	qs := GenerateQuestions(CandidateProfile{}, JobRequirements{}, 1, set)
	if len(qs) != 1 {
		t.Fatalf("questions = %d, want 1 (limit)", len(qs))
	}
	if qs[0].Prompt != llmQuestions()[0].Prompt {
		t.Fatalf("limit dropped the wrong question: %q", qs[0].Prompt)
	}
}

// Fallback selection: absent / empty / unusable stored sets must land on the
// deterministic templates, never on an empty interview.
func TestFallbackToTemplatesWhenSetUnusable(t *testing.T) {
	cv := CandidateProfile{Skills: []string{"Go"}}
	job := JobRequirements{Title: "Backend Engineer", RequiredSkills: []string{"Go", "Kubernetes"}}
	baseline := GenerateQuestions(cv, job, 5)
	if len(baseline) == 0 {
		t.Fatal("template baseline empty")
	}

	unusable := map[string]*jobdomain.QuestionSet{
		"nil set":          nil,
		"zero set":         {},
		"no questions":     storedSet(),
		"missing framing":  storedSet(jobdomain.Question{Category: jobdomain.QuestionCategoryTechnical, Prompt: "bare prompt"}),
		"unknown category": storedSet(jobdomain.Question{Category: "trivia", Prompt: "p", Context: "c", Expectation: "e"}),
	}
	for name, set := range unusable {
		t.Run(name, func(t *testing.T) {
			qs := GenerateQuestions(cv, job, 5, set)
			if len(qs) != len(baseline) {
				t.Fatalf("questions = %d, want template baseline %d", len(qs), len(baseline))
			}
			for i := range qs {
				if qs[i].Prompt != baseline[i].Prompt {
					t.Fatalf("question %d = %q, want template %q", i, qs[i].Prompt, baseline[i].Prompt)
				}
			}
		})
	}
}

// Bias filter is the LAST gate regardless of source — an LLM question probing
// a protected class never reaches a candidate.
func TestBiasFilterGatesStoredLLMQuestions(t *testing.T) {
	biased := jobdomain.Question{
		Category:    jobdomain.QuestionCategoryBehavioral,
		Prompt:      "Are you married, and do you have children at home?",
		Context:     "Framing that should not save a biased prompt.",
		Expectation: "Nothing — this question must be dropped.",
	}
	set := storedSet(append(llmQuestions(), biased)...)

	qs := GenerateQuestions(CandidateProfile{}, JobRequirements{}, 10, set)
	if len(qs) != 2 {
		t.Fatalf("questions = %d, want 2 (biased one dropped)", len(qs))
	}
	for _, q := range qs {
		if IsBiased(q.Prompt) {
			t.Fatalf("biased question survived: %q", q.Prompt)
		}
	}
}

// If EVERY stored question is biased the interview must still happen — the
// safe deterministic templates take over.
func TestAllBiasedStoredSetFallsBackToTemplates(t *testing.T) {
	set := storedSet(jobdomain.Question{
		Category:    jobdomain.QuestionCategoryBehavioral,
		Prompt:      "How old are you?",
		Context:     "c",
		Expectation: "e",
	})
	cv := CandidateProfile{Skills: []string{"Go"}}
	job := JobRequirements{Title: "Backend Engineer", RequiredSkills: []string{"Go"}}

	qs := GenerateQuestions(cv, job, 5, set)
	if len(qs) == 0 {
		t.Fatal("fell back to nothing — interview would start with zero questions")
	}
	for _, q := range qs {
		if IsBiased(q.Prompt) {
			t.Fatalf("biased question in fallback: %q", q.Prompt)
		}
	}
}

// Templates are descriptive too: same rendering contract as LLM output, so
// resume/replay shows identical framing whichever source was used.
func TestTemplateQuestionsCarryDescriptiveFields(t *testing.T) {
	cv := CandidateProfile{Skills: []string{"Go"}}
	job := JobRequirements{Title: "Backend Engineer", RequiredSkills: []string{"Go", "Kubernetes"}}
	for _, q := range GenerateQuestions(cv, job, 5) {
		if q.Context == "" || q.Expectation == "" {
			t.Fatalf("template question not self-explanatory: %+v", q)
		}
	}
}

// I6 read gate: a clean Prompt does not rescue biased framing — the stored-set
// gate must drop questions whose Context/Expectation/Skill carry biased or
// protected-class content, because those fields reach the candidate verbatim.
func TestStoredSetDropsBiasedFramingFields(t *testing.T) {
	cases := []struct {
		name        string
		context     string
		expectation string
		skill       string
	}{
		{name: "biased context", context: "We prefer candidates without children for on-call."},
		{name: "biased expectation", expectation: "Candidate should describe how a pregnancy would affect shift availability."},
		{name: "biased skill", skill: "church administration"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			biased := jobdomain.Question{
				Category:    jobdomain.QuestionCategoryTechnical,
				Prompt:      "Walk me through a production incident you resolved.",
				Context:     tc.context,
				Expectation: tc.expectation,
				Skill:       tc.skill,
			}
			if biased.Context == "" {
				biased.Context = "Incident response is part of this role's day-to-day."
			}
			if biased.Expectation == "" {
				biased.Expectation = "A clear incident timeline, mitigation steps, and follow-ups."
			}
			if biased.Skill == "" {
				biased.Skill = "postgresql"
			}
			if !IsBiased(biased.Context) && !IsBiased(biased.Expectation) && !IsBiased(biased.Skill) {
				// sanity: the fixture itself must trip the rail, else the test
				// proves nothing when the gate is widened.
				t.Fatalf("fixture %q does not trip IsBiased", tc.name)
			}
			set := storedSet(append(llmQuestions(), biased)...)
			qs := GenerateQuestions(CandidateProfile{}, JobRequirements{}, 10, set)
			if len(qs) != 2 {
				t.Fatalf("questions = %d, want 2 (framing-biased one dropped)", len(qs))
			}
			for _, q := range qs {
				if IsBiased(q.Context) || IsBiased(q.Expectation) || IsBiased(q.Skill) || IsBiased(q.Prompt) {
					t.Fatalf("question with biased framing survived: %+v", q)
				}
			}
		})
	}
}

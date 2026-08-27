package service

import (
	"sort"
	"strings"

	jobdomain "github.com/intivai/backend/internal/job/domain"
)

// Question categories per the docs (technical, behavioral, situational).
const (
	CategoryTechnical   = "technical"
	CategoryBehavioral  = "behavioral"
	CategorySituational = "situational"
)

type CandidateProfile struct {
	Skills          []string
	ExperienceYears float64
	Education       string
	Certifications  []string
	Summary         string
}

type JobRequirements struct {
	Title          string
	Description    string
	RequiredSkills []string
}

// Question — self-explanatory by contract: the prompt the candidate hears plus
// the framing that makes it defensible.
//
//   - Context     — why this question is asked for this role.
//   - Expectation — what a good answer demonstrates.
//
// Prompt keeps the "content" wire key: the interview transcript and the WS
// protocol already persist/emit questions under that name.
type Question struct {
	Category    string `json:"category"`
	Prompt      string `json:"content"`
	Context     string `json:"context,omitempty"`
	Expectation string `json:"expectation,omitempty"`
	Skill       string `json:"skill,omitempty"` // related skill for gap/depth questions
	Priority    int    `json:"priority"`        // 1=gap, 2=depth, 3=general
}

// GenerateQuestions resolves the interview question set for one candidate.
//
// Source order (decision D5 — "replace templates with LLM generation;
// deterministic templates demote to offline fallback only"):
//
//  1. stored LLM set — generated ONCE per job at publish time and passed in
//     here by the application layer. Used verbatim: every candidate for the
//     job hears the same questions and the interview start never waits on a
//     provider.
//  2. deterministic templates — CV-gap driven, used when no usable set exists
//     (generation failed, job pre-dates the feature, offline dev).
//
// `stored` is variadic so the fallback path stays a plain three-argument call;
// at most the first non-nil usable set is honoured. IsBiased gates EVERY
// question that leaves this function, whatever produced it.
func GenerateQuestions(cv CandidateProfile, job JobRequirements, limit int, stored ...*jobdomain.QuestionSet) []Question {
	if limit <= 0 {
		limit = 5
	}
	// A stored set that is entirely biased (or unusable) falls through to the
	// templates: an interview with zero questions is never an acceptable
	// outcome of a bad generation.
	if qs := fromStoredSet(firstUsableSet(stored), limit); len(qs) > 0 {
		return qs
	}
	return templateQuestions(cv, job, limit)
}

// firstUsableSet picks the first set that passes the stored contract (version +
// descriptive fields). Everything else means "templates".
func firstUsableSet(sets []*jobdomain.QuestionSet) *jobdomain.QuestionSet {
	for _, s := range sets {
		if s.Usable() {
			return s
		}
	}
	return nil
}

// fromStoredSet maps the job-owned stored questions onto interview questions,
// preserving stored order (that order IS the per-job determinism) and applying
// the bias filter as the last gate.
func fromStoredSet(set *jobdomain.QuestionSet, limit int) []Question {
	if set == nil {
		return nil
	}
	out := make([]Question, 0, len(set.Questions))
	for _, q := range set.Questions {
		if len(out) >= limit {
			break
		}
		// Last-line defense (I6): a provider-authored question probing a
		// protected class is dropped even though it was validated at
		// generation time. EVERY candidate-visible field is gated — a clean
		// Prompt does not rescue biased Context/Expectation/Skill framing.
		if IsBiased(q.Prompt) || IsBiased(q.Context) || IsBiased(q.Expectation) || IsBiased(q.Skill) {
			continue
		}
		out = append(out, Question{
			Category:    q.Category,
			Prompt:      q.Prompt,
			Context:     q.Context,
			Expectation: q.Expectation,
			Skill:       q.Skill,
			Priority:    q.Priority,
		})
	}
	return out
}

// templateQuestions builds the deterministic fallback set from CV gaps:
//  1. missing required skills → gap questions (highest priority)
//  2. claimed required skills → depth verification
//  3. general behavioral/situational to fill the limit
//
// Deterministic ordering; all output passes the bias filter and carries the
// same descriptive framing as LLM output, so resume/replay renders identically
// whichever source produced the question.
func templateQuestions(cv CandidateProfile, job JobRequirements, limit int) []Question {
	candidate := lowerSet(cv.Skills)
	requiredSet := lowerSet(job.RequiredSkills)

	gaps := difference(requiredSet, candidate)
	depth := intersection(candidate, requiredSet)

	questions := []Question{}
	for _, skill := range gaps {
		questions = append(questions, Question{
			Category:    CategoryTechnical,
			Prompt:      "What experience do you have with " + skill + "?",
			Context:     "This role requires " + skill + ", and your CV does not evidence it — a missing keyword is not proof of a missing skill.",
			Expectation: "Concrete hands-on work with " + skill + ", or a straight answer about the gap and how you would close it.",
			Skill:       skill,
			Priority:    1,
		})
	}
	for _, skill := range depth {
		questions = append(questions, Question{
			Category:    CategoryTechnical,
			Prompt:      "Describe a project where you applied " + skill + " in production.",
			Context:     "Your CV claims " + skill + " and this role depends on it, so we verify depth beyond the keyword.",
			Expectation: "Specifics: what you built, the decisions you made, the tradeoffs you accepted, and the outcome.",
			Skill:       skill,
			Priority:    2,
		})
	}

	questions = append(questions, generalPool(job)...)

	// Bias filter — generated templates are safe, but defense in depth.
	filtered := questions[:0]
	for _, q := range questions {
		if !IsBiased(q.Prompt) {
			filtered = append(filtered, q)
		}
	}
	questions = filtered

	// Deterministic order: priority, then category, then prompt.
	sort.SliceStable(questions, func(i, j int) bool {
		if questions[i].Priority != questions[j].Priority {
			return questions[i].Priority < questions[j].Priority
		}
		if questions[i].Category != questions[j].Category {
			return questions[i].Category < questions[j].Category
		}
		return questions[i].Prompt < questions[j].Prompt
	})

	// Diversity: alternate categories within the limit, keep gap/depth first.
	kept := append([]Question{}, questions[:min(len(questions), priorityCount(questions))]...)
	keptPrompts := map[string]bool{}
	for _, q := range kept {
		keptPrompts[q.Prompt] = true
	}
	rest := questions[len(kept):]
	lastCat := ""
	if len(kept) > 0 {
		lastCat = kept[len(kept)-1].Category
	}
	for _, q := range rest {
		if len(kept) >= limit {
			break
		}
		if q.Category == lastCat || keptPrompts[q.Prompt] {
			continue
		}
		kept = append(kept, q)
		keptPrompts[q.Prompt] = true
		lastCat = q.Category
	}
	if len(kept) < limit {
		for _, q := range rest {
			if len(kept) >= limit {
				break
			}
			if keptPrompts[q.Prompt] {
				continue
			}
			kept = append(kept, q)
			keptPrompts[q.Prompt] = true
		}
	}
	if len(kept) > limit {
		kept = kept[:limit]
	}
	return kept
}

func priorityCount(questions []Question) int {
	n := 0
	for _, q := range questions {
		if q.Priority <= 2 {
			n++
		}
	}
	return n
}

func generalPool(job JobRequirements) []Question {
	title := job.Title
	if title == "" {
		title = "the role"
	}
	return []Question{
		{
			Category:    CategoryBehavioral,
			Prompt:      "Tell me about a time you disagreed with a teammate and how you resolved it.",
			Context:     "Technical decisions here are made in the open, so healthy disagreement is part of the work.",
			Expectation: "You listen, argue with evidence, and commit to the decision once it is made.",
			Priority:    3,
		},
		{
			Category:    CategoryBehavioral,
			Prompt:      "Describe a project that missed a deadline — what happened and what did you learn?",
			Context:     "Delivery slips happen; how you read the signals and react is what we can learn from.",
			Expectation: "Honest ownership, the early warning you missed, and the concrete change you made after.",
			Priority:    3,
		},
		{
			Category:    CategorySituational,
			Prompt:      "You receive two urgent tasks with the same deadline. How do you prioritize?",
			Context:     "Competing priorities are routine, and we want to see how you decide without an escalation.",
			Expectation: "An explicit tradeoff: impact, dependencies, who you tell, and what you drop.",
			Priority:    3,
		},
		{
			Category:    CategorySituational,
			Prompt:      "A requirement for " + title + " changes mid-sprint. How do you handle it?",
			Context:     "Requirements for " + title + " do move, so adapting without derailing the sprint matters.",
			Expectation: "You confirm the change, re-scope deliberately, and keep everyone informed.",
			Priority:    3,
		},
	}
}

func lowerSet(items []string) map[string]bool {
	out := make(map[string]bool, len(items))
	for _, i := range items {
		if v := strings.ToLower(strings.TrimSpace(i)); v != "" {
			out[v] = true
		}
	}
	return out
}

func difference(required, candidate map[string]bool) []string {
	out := []string{}
	for r := range required {
		if !candidate[r] {
			out = append(out, r)
		}
	}
	sort.Strings(out)
	return out
}

func intersection(candidate, required map[string]bool) []string {
	out := []string{}
	for c := range candidate {
		if required[c] {
			out = append(out, c)
		}
	}
	sort.Strings(out)
	return out
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

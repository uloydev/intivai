package service

import (
	"sort"
	"strings"
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

type Question struct {
	Category string `json:"category"`
	Prompt   string `json:"content"`
	Skill    string `json:"skill,omitempty"` // related skill for gap/depth questions
	Priority int    `json:"priority"`        // 1=gap, 2=depth, 3=general
}

// GenerateQuestions builds an interview question set from CV gaps:
//  1. missing required skills → gap questions (highest priority)
//  2. claimed required skills → depth verification
//  3. general behavioral/situational to fill the limit
//
// Deterministic ordering; all output passes the bias filter.
func GenerateQuestions(cv CandidateProfile, job JobRequirements, limit int) []Question {
	if limit <= 0 {
		limit = 5
	}
	candidate := lowerSet(cv.Skills)
	requiredSet := lowerSet(job.RequiredSkills)

	gaps := difference(requiredSet, candidate)
	depth := intersection(candidate, requiredSet)

	questions := []Question{}
	for _, skill := range gaps {
		questions = append(questions, Question{
			Category: CategoryTechnical,
			Prompt:   "What experience do you have with " + skill + "?",
			Skill:    skill,
			Priority: 1,
		})
	}
	for _, skill := range depth {
		questions = append(questions, Question{
			Category: CategoryTechnical,
			Prompt:   "Describe a project where you applied " + skill + " in production.",
			Skill:    skill,
			Priority: 2,
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
		{Category: CategoryBehavioral, Prompt: "Tell me about a time you disagreed with a teammate and how you resolved it.", Priority: 3},
		{Category: CategoryBehavioral, Prompt: "Describe a project that missed a deadline — what happened and what did you learn?", Priority: 3},
		{Category: CategorySituational, Prompt: "You receive two urgent tasks with the same deadline. How do you prioritize?", Priority: 3},
		{Category: CategorySituational, Prompt: "A requirement for " + title + " changes mid-sprint. How do you handle it?", Priority: 3},
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

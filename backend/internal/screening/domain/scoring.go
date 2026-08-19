package domain

import (
	"strings"
)

// DefaultWeights — canonical from the architecture docs (Phases §Scoring Engine).
func DefaultWeights() map[string]float64 {
	return map[string]float64{
		"skills_match":     0.35,
		"experience_years": 0.20,
		"semantic_match":   0.25,
		"education":        0.10,
		"certifications":   0.10,
	}
}

type ResumeData struct {
	Skills          []string `json:"skills"`
	ExperienceYears float64  `json:"experience_years"`
	Education       string   `json:"education"`
	Certifications  []string `json:"certifications"`
	Summary         string   `json:"summary"`
}

type JobInfo struct {
	RequiredSkills    []string
	MinExperience     int
	MinScoreToProceed float64
	ScoringWeights    map[string]float64 // job override (partial)
}

type OrgInfo struct {
	ScoringWeights    map[string]float64 // org override (partial)
	MinScoreToProceed float64
}

type ScoreResult struct {
	Total     float64            `json:"total"`
	Breakdown map[string]float64 `json:"breakdown"`
	Passed    bool               `json:"passed"`
}

// Score computes the weighted score. SemanticScore is provided by the caller
// (keyword overlap now; embeddings in M2.5).
func Score(resume ResumeData, job JobInfo, org OrgInfo, semanticScore float64) ScoreResult {
	weights := resolveWeights(job.ScoringWeights, org.ScoringWeights)

	skills := scoreSkills(resume.Skills, job.RequiredSkills)
	experience := scoreExperience(resume.ExperienceYears, job.MinExperience)
	semantic := clamp(semanticScore)
	education := scoreEducation(resume.Education)
	certifications := scoreCertifications(resume.Certifications)

	breakdown := map[string]float64{
		"skills_match":     skills,
		"experience_years": experience,
		"semantic_match":   semantic,
		"education":        education,
		"certifications":   certifications,
	}
	total := 0.0
	for key, weight := range weights {
		total += weight * breakdown[key]
	}

	minScore := job.MinScoreToProceed
	if minScore == 0 {
		minScore = org.MinScoreToProceed
	}
	if minScore == 0 {
		minScore = 50
	}

	total = round(total * 100) // weights sum to 1.0 → normalize to 0-100 scale
	return ScoreResult{Total: total, Breakdown: breakdown, Passed: total >= minScore}
}

// resolveWeights merges partial overrides: job > org > defaults, per key.
func resolveWeights(jobOver, orgOver map[string]float64) map[string]float64 {
	out := DefaultWeights()
	for k, v := range orgOver {
		out[k] = v
	}
	for k, v := range jobOver {
		out[k] = v
	}
	return out
}

func scoreSkills(candidate, required []string) float64 {
	if len(required) == 0 {
		return 1.0
	}
	req := toSet(required)
	if len(req) == 0 {
		return 1.0
	}
	matched := 0
	for _, s := range candidate {
		if req[strings.ToLower(strings.TrimSpace(s))] {
			matched++
		}
	}
	if matched > len(req) {
		return 1.0
	}
	return float64(matched) / float64(len(req))
}

func scoreExperience(years float64, min int) float64 {
	if years < 0 {
		return 0
	}
	if min <= 0 {
		return 1.0
	}
	ratio := years / float64(min)
	if ratio > 1 {
		return 1.0
	}
	return ratio
}

func scoreEducation(education string) float64 {
	e := strings.ToLower(education)
	switch {
	case strings.Contains(e, "phd"), strings.Contains(e, "doctorate"):
		return 1.0
	case strings.Contains(e, "master"):
		return 0.8
	case strings.Contains(e, "bachelor"), strings.Contains(e, "undergraduate"):
		return 0.6
	case strings.Contains(e, "high school"):
		return 0.2
	case strings.Contains(e, "diploma"), strings.Contains(e, "associate"):
		return 0.4
	default:
		return 0.5
	}
}

func scoreCertifications(certs []string) float64 {
	if len(certs) > 0 {
		return 1.0
	}
	return 0
}

func toSet(items []string) map[string]bool {
	out := make(map[string]bool, len(items))
	for _, i := range items {
		if v := strings.ToLower(strings.TrimSpace(i)); v != "" {
			out[v] = true
		}
	}
	return out
}

func clamp(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func round(v float64) float64 {
	return float64(int(v*1000+0.5)) / 1000
}

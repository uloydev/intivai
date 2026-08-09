package domain

import (
	"testing"
)

func TestScoreMath(t *testing.T) {
	resume := ResumeData{
		Skills:          []string{"Go", "PostgreSQL", "Kubernetes"},
		ExperienceYears: 5,
		Education:       "Master of Science",
		Certifications:  []string{"AWS SA"},
	}
	job := JobInfo{
		RequiredSkills: []string{"Go", "PostgreSQL", "Kubernetes"},
		MinExperience:  3,
	}
	result := Score(resume, job, OrgInfo{}, 1.0)

	want := 0.35*1.0 + 0.20*1.0 + 0.25*1.0 + 0.10*0.8 + 0.10*1.0
	if result.Total != want*100 {
		t.Fatalf("total = %v, want %v", result.Total, want)
	}
	if !result.Passed {
		t.Fatal("expected pass with high score")
	}
}

func TestScoreSkillOverlap(t *testing.T) {
	resume := ResumeData{Skills: []string{"Go"}}
	job := JobInfo{RequiredSkills: []string{"Go", "PostgreSQL", "Kubernetes"}, MinExperience: 2}
	result := Score(resume, job, OrgInfo{}, 0)

	if got := result.Breakdown["skills_match"]; got != 1.0/3.0 {
		t.Fatalf("skills_match = %v, want %v", got, 1.0/3.0)
	}
}

func TestScoreExperienceCap(t *testing.T) {
	resume := ResumeData{ExperienceYears: 10}
	job := JobInfo{MinExperience: 3}
	result := Score(resume, job, OrgInfo{}, 0)
	if got := result.Breakdown["experience_years"]; got != 1.0 {
		t.Fatalf("experience = %v, want 1.0", got)
	}
}

func TestScoreThresholdFromOrg(t *testing.T) {
	resume := ResumeData{Skills: []string{"Go"}}
	job := JobInfo{RequiredSkills: []string{"Go"}}
	result := Score(resume, job, OrgInfo{MinScoreToProceed: 50}, 1.0)
	if !result.Passed {
		t.Fatal("expected pass with org threshold 10")
	}
}

func TestWeightResolutionJobOverridesOrg(t *testing.T) {
	resume := ResumeData{Skills: []string{"Go"}}
	job := JobInfo{
		RequiredSkills: []string{"Go"},
		ScoringWeights: map[string]float64{"skills_match": 1.0, "experience_years": 0, "semantic_match": 0, "education": 0, "certifications": 0},
	}
	org := OrgInfo{ScoringWeights: map[string]float64{"skills_match": 0.1}}
	result := Score(resume, job, org, 0)
	if result.Total != 100 {
		t.Fatalf("total = %v, want 100 (job weight wins, 0-100 scale)", result.Total)
	}
}

func TestScoreNoRequiredSkillsNeutral(t *testing.T) {
	resume := ResumeData{}
	job := JobInfo{}
	result := Score(resume, job, OrgInfo{}, 1.0)
	if got := result.Breakdown["skills_match"]; got != 1.0 {
		t.Fatalf("skills_match = %v, want 1.0 (neutral when no required skills)", got)
	}
}

func TestSemanticScore(t *testing.T) {
	cand := "senior software engineer experienced with Go and PostgreSQL and distributed systems"
	job := "we need Go and PostgreSQL engineers for distributed systems work"
	score := SemanticScore(cand, job)
	if score < 0.5 {
		t.Fatalf("semantic = %v, want >= 0.5 for strong overlap", score)
	}
	low := SemanticScore("marketing copywriter for fashion brands", job)
	if low != 0 {
		t.Fatalf("semantic = %v, want 0 for no overlap", low)
	}
}

func TestEducationLevels(t *testing.T) {
	cases := map[string]float64{
		"PhD in Computer Science": 1.0,
		"Master of Business":      0.8,
		"Bachelor of Engineering": 0.6,
		"Higher National Diploma": 0.4,
		"high school diploma":     0.2,
		"":                        0,
	}
	for input, want := range cases {
		if got := scoreEducation(input); got != want {
			t.Errorf("scoreEducation(%q) = %v, want %v", input, got, want)
		}
	}
}

package service

import (
	"strings"
	"testing"
)

func TestGapQuestionsFirst(t *testing.T) {
	cv := CandidateProfile{Skills: []string{"Go"}}
	job := JobRequirements{RequiredSkills: []string{"Go", "Kubernetes", "PostgreSQL"}}
	qs := GenerateQuestions(cv, job, 5)

	if len(qs) == 0 {
		t.Fatal("no questions generated")
	}
	// Both gaps must appear as technical gap questions before anything else.
	if qs[0].Priority != 1 || !strings.Contains(strings.ToLower(qs[0].Prompt), "kubernetes") {
		t.Fatalf("first question = %+v, want Kubernetes gap", qs[0])
	}
	hasK8s, hasPg := false, false
	for _, q := range qs {
		lower := strings.ToLower(q.Prompt)
		if strings.Contains(lower, "kubernetes") {
			hasK8s = true
		}
		if strings.Contains(lower, "postgresql") {
			hasPg = true
		}
	}
	if !hasK8s || !hasPg {
		t.Fatalf("gap questions missing: k8s=%v pg=%v", hasK8s, hasPg)
	}
}

func TestDepthVerificationForClaimedSkills(t *testing.T) {
	cv := CandidateProfile{Skills: []string{"Go"}}
	job := JobRequirements{RequiredSkills: []string{"Go"}}
	qs := GenerateQuestions(cv, job, 5)

	foundDepth := false
	for _, q := range qs {
		if q.Skill == "go" && q.Priority == 2 {
			foundDepth = true
		}
	}
	if !foundDepth {
		t.Fatalf("no depth question for claimed skill: %+v", qs)
	}
}

func TestLimitAndCategories(t *testing.T) {
	cv := CandidateProfile{Skills: []string{"Go", "Kubernetes"}}
	job := JobRequirements{RequiredSkills: []string{"Go", "Kubernetes"}, Title: "Backend Engineer"}
	qs := GenerateQuestions(cv, job, 4)

	if len(qs) > 4 {
		t.Fatalf("limit exceeded: %d", len(qs))
	}
	cats := map[string]bool{}
	for _, q := range qs {
		cats[q.Category] = true
	}
	if !cats[CategoryTechnical] || !cats[CategoryBehavioral] || !cats[CategorySituational] {
		t.Fatalf("category diversity missing: %v", cats)
	}
}

func TestNoBiasInGeneratedQuestions(t *testing.T) {
	cv := CandidateProfile{Skills: []string{}}
	job := JobRequirements{RequiredSkills: []string{}}
	qs := GenerateQuestions(cv, job, 20)
	for _, q := range qs {
		if IsBiased(q.Prompt) {
			t.Fatalf("biased question generated: %q", q.Prompt)
		}
	}
}

func TestDeterministicOutput(t *testing.T) {
	cv := CandidateProfile{Skills: []string{"Go"}}
	job := JobRequirements{RequiredSkills: []string{"Go", "Kubernetes"}}
	a := GenerateQuestions(cv, job, 5)
	b := GenerateQuestions(cv, job, 5)
	if len(a) != len(b) {
		t.Fatalf("length differs: %d vs %d", len(a), len(b))
	}
	for i := range a {
		if a[i].Prompt != b[i].Prompt {
			t.Fatalf("order differs at %d: %q vs %q", i, a[i].Prompt, b[i].Prompt)
		}
	}
}

func TestNoDuplicatePrompts(t *testing.T) {
	cv := CandidateProfile{Skills: []string{"Go"}}
	job := JobRequirements{RequiredSkills: []string{"Go", "Kubernetes"}}
	qs := GenerateQuestions(cv, job, 10)
	seen := map[string]bool{}
	for _, q := range qs {
		if seen[q.Prompt] {
			t.Fatalf("duplicate prompt: %q", q.Prompt)
		}
		seen[q.Prompt] = true
	}
}

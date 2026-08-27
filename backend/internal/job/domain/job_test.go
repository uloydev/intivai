package domain

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	sharederrors "github.com/intivai/backend/internal/shared/errors"
)

func TestNewJobValidation(t *testing.T) {
	if _, err := NewJob(uuid.New(), "", "d", nil, 0); err == nil {
		t.Fatal("empty title accepted")
	}
	if _, err := NewJob(uuid.New(), "t", "", nil, 0); err == nil {
		t.Fatal("empty description accepted")
	}
	if _, err := NewJob(uuid.New(), "t", "d", nil, -1); err == nil {
		t.Fatal("negative experience accepted")
	}
	job, err := NewJob(uuid.New(), "t", "d", []string{"Go"}, 3)
	if err != nil {
		t.Fatal(err)
	}
	if job.Status != StatusActive {
		t.Fatalf("status = %q, want active", job.Status)
	}
}

func TestValidateJobFieldsStatus(t *testing.T) {
	if err := ValidateJobFields("t", "d", 0, "bogus"); err == nil {
		t.Fatal("invalid status accepted")
	}
	if err := ValidateJobFields("t", "d", 0, StatusArchived); err != nil {
		t.Fatal(err)
	}
}

// fullWeights returns a complete weight map summing exactly to 1.
func fullWeights() map[string]float64 {
	return map[string]float64{
		"skills_match":     0.35,
		"experience_years": 0.20,
		"semantic_match":   0.25,
		"education":        0.10,
		"certifications":   0.10,
	}
}

func assertWeightCode(t *testing.T, err error, code string) {
	t.Helper()
	var de *sharederrors.DomainError
	if !errors.As(err, &de) {
		t.Fatalf("err = %v, want DomainError %s", err, code)
	}
	if de.Code != code {
		t.Fatalf("code = %q, want %q", de.Code, code)
	}
}

func TestSetScoringWeights(t *testing.T) {
	job, _ := NewJob(uuid.New(), "t", "d", nil, 0)

	// Nil and empty mean "keep defaults" — field omitted upstream.
	if err := job.SetScoringWeights(nil); err != nil {
		t.Fatal(err)
	}
	if err := job.SetScoringWeights(map[string]float64{}); err != nil {
		t.Fatal(err)
	}
	if job.ScoringWeights != nil {
		t.Fatal("empty input must not set weights")
	}

	// Complete map summing to 1 is accepted; zero-valued dimensions legal.
	full := fullWeights()
	full["education"] = 0
	full["certifications"] = 0
	full["skills_match"] = 0.55
	if err := job.SetScoringWeights(full); err != nil {
		t.Fatal(err)
	}
	if job.ScoringWeights["skills_match"] != 0.55 {
		t.Fatal("weights not stored")
	}

	// Partial maps rejected — silent merge with defaults hides math from HR.
	assertWeightCode(t, job.SetScoringWeights(map[string]float64{"skills_match": 0.5}), "WEIGHTS_INCOMPLETE")

	// Sum tolerance ±0.01 around 1.0.
	bad := fullWeights()
	bad["semantic_match"] = 0.30 // sums to 1.05
	assertWeightCode(t, job.SetScoringWeights(bad), "WEIGHTS_SUM_INVALID")

	worst := fullWeights()
	worst["semantic_match"] = 0.10 // sums to 0.85
	assertWeightCode(t, job.SetScoringWeights(worst), "WEIGHTS_SUM_INVALID")

	ok := fullWeights()
	ok["semantic_match"] = 0.26 // sums to 1.01 — inside tolerance
	if err := job.SetScoringWeights(ok); err != nil {
		t.Fatal(err)
	}

	// Unknown names and out-of-range values stay invalid.
	badName := fullWeights()
	badName["nope"] = badName["semantic_match"]
	delete(badName, "semantic_match")
	assertWeightCode(t, job.SetScoringWeights(badName), "WEIGHTS_INVALID")

	badRange := fullWeights()
	badRange["education"] = 1.5
	assertWeightCode(t, job.SetScoringWeights(badRange), "WEIGHTS_INVALID")

	negative := fullWeights()
	negative["education"] = -0.1
	negative["certifications"] = 0.2
	assertWeightCode(t, job.SetScoringWeights(negative), "WEIGHTS_INVALID")
}

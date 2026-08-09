package domain

import (
	"testing"

	"github.com/google/uuid"
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

func TestSetScoringWeights(t *testing.T) {
	job, _ := NewJob(uuid.New(), "t", "d", nil, 0)
	if err := job.SetScoringWeights(map[string]float64{"skills_match": 0.5}); err != nil {
		t.Fatal(err)
	}
	if err := job.SetScoringWeights(map[string]float64{"skills_match": 1.5}); err == nil {
		t.Fatal("weight > 1 accepted")
	}
	if err := job.SetScoringWeights(map[string]float64{"nope": 0.5}); err == nil {
		t.Fatal("unknown weight name accepted")
	}
}

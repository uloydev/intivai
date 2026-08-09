package domain

import (
	"testing"

	"github.com/google/uuid"
)

func TestNewCandidateValidation(t *testing.T) {
	if _, err := NewCandidate(uuid.New(), "", "e@x.io"); err == nil {
		t.Fatal("empty name accepted")
	}
	c, err := NewCandidate(uuid.New(), "Jane", "j@x.io")
	if err != nil {
		t.Fatal(err)
	}
	if c.Status != StatusNew {
		t.Fatalf("status = %q, want new", c.Status)
	}
}

func TestCandidateStatesValid(t *testing.T) {
	for _, s := range []string{StatusNew, StatusParsing, StatusParsed, StatusExtracting, StatusExtracted, StatusFailedOCR, StatusFailedExtract} {
		if s == "" {
			t.Fatal("empty status constant")
		}
	}
}

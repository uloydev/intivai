package domain

import (
	"encoding/json"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/intivai/backend/internal/shared/domain"
	"github.com/intivai/backend/internal/shared/errors"
)

const (
	StatusActive   = "active"
	StatusArchived = "archived"
)

// Job is a job description owned by a tenant. Scoring overrides are optional:
// job > org > global defaults (resolved at score time).
type Job struct {
	domain.Entity
	OrgID             uuid.UUID
	Title             string
	Description       string
	Location          string
	EmploymentType    string
	SalaryMin         *int
	SalaryMax         *int
	Currency          string
	RequiredSkills    []string
	MinExperience     int
	Responsibilities  []string
	Requirements      []string
	NiceToHaves       []string
	Benefits          []string
	ScoringWeights    map[string]float64
	MinScoreToProceed *float64
	Status            string
	ProctoringMode    string
	IsPublished       bool
	Rubric            json.RawMessage
	// QuestionSetError carries the terminal generation failure (D9) so the
	// API can surface it; "" = no recorded failure. Written only by the
	// question worker's column-scoped error store.
	QuestionSetError string
}

func NewJob(orgID uuid.UUID, title, description string, requiredSkills []string, minExperience int) (*Job, error) {
	if err := ValidateJobFields(title, description, minExperience, StatusActive); err != nil {
		return nil, err
	}
	return &Job{
		Entity:           domain.Entity{ID: domain.NewID(), CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()},
		OrgID:            orgID,
		Title:            title,
		Description:      description,
		Location:         "Remote",
		EmploymentType:   "Full-time",
		Currency:         "USD",
		RequiredSkills:   requiredSkills,
		MinExperience:    minExperience,
		Responsibilities: []string{},
		Requirements:     []string{},
		NiceToHaves:      []string{},
		Benefits:         []string{},
		Status:           StatusActive,
		IsPublished:      false,
	}, nil
}

// ValidateJobFields — shared by create and update paths.
func ValidateJobFields(title, description string, minExperience int, status string) error {
	if title == "" {
		return errors.NewDomainError("JOB_TITLE_REQUIRED", "job title is required")
	}
	if description == "" {
		return errors.NewDomainError("JOB_DESC_REQUIRED", "job description is required")
	}
	if minExperience < 0 {
		return errors.NewDomainError("JOB_EXP_INVALID", "min experience cannot be negative")
	}
	if status != "" && status != StatusActive && status != StatusArchived {
		return errors.NewDomainError("JOB_STATUS_INVALID", "job status must be active or archived")
	}
	return nil
}

// weightDimensions — the fixed scoring taxonomy. A scoring_weights payload
// must be COMPLETE (every dimension present) when provided: partial maps
// silently merge with org/global defaults and hide the effective math from
// HR users.
var weightDimensions = []string{"skills_match", "experience_years", "semantic_match", "education", "certifications"}

func (j *Job) SetScoringWeights(raw map[string]float64) error {
	if len(raw) == 0 {
		return nil // field omitted upstream — keep job > org > default resolution untouched
	}
	// Typos first — an unknown key is the user's actual problem even when a
	// dimension is also missing.
	for k := range raw {
		if !validWeightName(k) {
			return errors.NewDomainError("WEIGHTS_INVALID", "invalid scoring weight: "+k)
		}
	}
	for _, dim := range weightDimensions {
		if _, ok := raw[dim]; !ok {
			return errors.NewDomainError("WEIGHTS_INCOMPLETE", "scoring_weights must include every dimension: "+strings.Join(weightDimensions, ", "))
		}
	}
	for _, v := range raw {
		if v < 0 || v > 1 {
			return errors.NewDomainError("WEIGHTS_INVALID", "invalid scoring weight: value out of range")
		}
	}
	sum := 0.0
	for _, v := range raw {
		sum += v
	}
	// Scale to integer percent before comparing: float64 accumulation must not
	// decide the ±0.01 boundary (1.01 - 1.0 == 0.010000000000000009).
	scaled := int(math.Round(sum * 100))
	if scaled < 100-1 || scaled > 100+1 {
		return errors.NewDomainError("WEIGHTS_SUM_INVALID", "scoring_weights must sum to 1.0 (±0.01)")
	}
	j.ScoringWeights = raw
	return nil
}

func validWeightName(k string) bool {
	switch k {
	case "skills_match", "experience_years", "semantic_match", "education", "certifications":
		return true
	}
	return false
}

func (j *Job) MarshalScoringWeights() (json.RawMessage, error) {
	if j.ScoringWeights == nil {
		return nil, nil
	}
	return json.Marshal(j.ScoringWeights)
}

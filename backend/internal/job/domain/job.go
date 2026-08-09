package domain

import (
	"encoding/json"
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
	RequiredSkills    []string
	MinExperience     int
	ScoringWeights    map[string]float64
	MinScoreToProceed *float64
	Status            string
}

func NewJob(orgID uuid.UUID, title, description string, requiredSkills []string, minExperience int) (*Job, error) {
	if err := ValidateJobFields(title, description, minExperience, StatusActive); err != nil {
		return nil, err
	}
	return &Job{
		Entity:         domain.Entity{ID: domain.NewID(), CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()},
		OrgID:          orgID,
		Title:          title,
		Description:    description,
		RequiredSkills: requiredSkills,
		MinExperience:  minExperience,
		Status:         StatusActive,
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

func (j *Job) SetScoringWeights(raw map[string]float64) error {
	if raw == nil {
		return nil
	}
	for k, v := range raw {
		if !validWeightName(k) || v < 0 || v > 1 {
			return errors.NewDomainError("INVALID_WEIGHT", "invalid scoring weight: "+k)
		}
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

func (j *Job) MarshalScoringWeights() json.RawMessage {
	if j.ScoringWeights == nil {
		return nil
	}
	b, _ := json.Marshal(j.ScoringWeights)
	return b
}

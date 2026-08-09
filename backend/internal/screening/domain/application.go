package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/intivai/backend/internal/shared/domain"
)

const (
	StatusScreening = "screening"
	StatusPassed    = "passed"
	StatusRejected  = "rejected"
)

type Application struct {
	domain.Entity
	OrgID           uuid.UUID
	CandidateID     uuid.UUID
	JobID           uuid.UUID
	CVScore         *float64
	ScoreBreakdown  json.RawMessage
	PassedScreening *bool
	Status          string
}

func NewApplication(orgID, candidateID, jobID uuid.UUID) *Application {
	return &Application{
		Entity:      domain.Entity{ID: domain.NewID(), CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()},
		OrgID:       orgID,
		CandidateID: candidateID,
		JobID:       jobID,
		Status:      StatusScreening,
	}
}

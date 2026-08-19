package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/intivai/backend/internal/shared/domain"
	"github.com/intivai/backend/internal/shared/errors"
)

// Candidate states — the async pipeline moves through these; failures are
// terminal states with error_message set (audit-visible).
const (
	StatusNew           = "new"
	StatusParsing       = "parsing"
	StatusParsed        = "parsed"
	StatusExtracting    = "extracting"
	StatusExtracted     = "extracted"
	StatusPendingReview = "pending_review"
	StatusFailedOCR     = "failed_ocr"
	StatusFailedExtract = "failed_extract"
)

type Candidate struct {
	domain.Entity
	OrgID        uuid.UUID
	Name         string
	Email        string
	CVPath       string
	CVRawText    string
	CVStructured json.RawMessage
	CVOCRMethod  string
	Status       string
	ErrorMessage string
	BatchID      *uuid.UUID
	ReviewToken  *string
}

func NewCandidate(orgID uuid.UUID, name, email string) (*Candidate, error) {
	if name == "" {
		return nil, errors.NewDomainError("CANDIDATE_NAME_REQUIRED", "candidate name is required")
	}
	return &Candidate{
		Entity: domain.Entity{ID: domain.NewID(), CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()},
		OrgID:  orgID,
		Name:   name,
		Email:  email,
		Status: StatusNew,
	}, nil
}

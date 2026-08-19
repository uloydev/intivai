package application

import (
	scrdomain "github.com/intivai/backend/internal/screening/domain"
)

// ResumeData — structured CV extraction (DeepSeek json_object output).
// Canonical type lives in the screening domain (the scoring engine consumes
// it); this package aliases it so the extract pipeline and the scorer always
// speak the same shape.
type ResumeData = scrdomain.ResumeData

type ParseCVPayload struct {
	OrgID       string `json:"org_id"`
	CandidateID string `json:"candidate_id"`
}

type ExtractCVPayload struct {
	OrgID       string `json:"org_id"`
	CandidateID string `json:"candidate_id"`
}

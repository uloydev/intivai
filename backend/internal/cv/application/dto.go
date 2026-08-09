package application

import (
	"github.com/google/uuid"
)

// ResumeData — structured CV extraction (DeepSeek json_object output).
type ResumeData struct {
	Skills          []string `json:"skills"`
	ExperienceYears float64  `json:"experience_years"`
	Education       string   `json:"education"`
	Certifications  []string `json:"certifications"`
	Summary         string   `json:"summary"`
}

type ParseCVPayload struct {
	OrgID       string `json:"org_id"`
	CandidateID string `json:"candidate_id"`
}

type ExtractCVPayload struct {
	OrgID       string `json:"org_id"`
	CandidateID string `json:"candidate_id"`
}

func mustUUID(s string) uuid.UUID {
	id, _ := uuid.Parse(s)
	return id
}

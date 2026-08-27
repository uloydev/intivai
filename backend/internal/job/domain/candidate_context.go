package domain

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	ctxdomain "github.com/intivai/backend/internal/context/domain"
	"github.com/intivai/backend/internal/shared/errors"
)

// MaxCandidateContextLength caps the per-job candidate Q&A context. It is
// inlined into the interview system prompt, so it must stay within a sane
// budget (mirrors the company-context prompt cap).
const MaxCandidateContextLength = 4000

// CandidateContext is the per-job recruiter-authored Q&A context (D2) merged
// into the interview system prompt alongside the org company context, version
// pinned at connect time so mid-interview edits can't change what a candidate
// hears.
type CandidateContext struct {
	JobID     uuid.UUID
	OrgID     uuid.UUID
	Content   string
	Version   int
	UpdatedAt time.Time
}

// NewCandidateContext validates and constructs a candidate context for a job.
// It rejects empty / over-long / injection-laden content before any write.
func NewCandidateContext(orgID, jobID uuid.UUID, content string) (*CandidateContext, error) {
	if err := ValidateCandidateContext(content); err != nil {
		return nil, err
	}
	return &CandidateContext{
		JobID:     jobID,
		OrgID:     orgID,
		Content:   strings.TrimSpace(content),
		Version:   1,
		UpdatedAt: time.Now().UTC(),
	}, nil
}

// ValidateCandidateContext runs the shared prompt-injection rail (tenant
// prompt + company context) so the per-job context can never smuggle
// instructions into the system prompt.
func ValidateCandidateContext(content string) error {
	if strings.TrimSpace(content) == "" {
		return errors.NewDomainError("CANDIDATE_CONTEXT_REQUIRED", "candidate context content is required")
	}
	if len(content) > MaxCandidateContextLength {
		return errors.NewDomainError("CANDIDATE_CONTEXT_TOO_LONG", "candidate context exceeds max length")
	}
	if ctxdomain.ContainsInjection(content) {
		return errors.NewDomainError("CANDIDATE_CONTEXT_INJECTION", "candidate context contains forbidden content")
	}
	return nil
}

// CandidateContextRepository — storage contract for per-job contexts.
// Implementations must run inside a tenant transaction (RLS applies).
type CandidateContextRepository interface {
	// Upsert writes the context for a job, bumping version on each save.
	// It returns the row's ACTUAL post-write version and updated_at (J12):
	// the ON CONFLICT branch increments version server-side, so a caller
	// that keeps its own copy would report a stale version.
	Upsert(ctx context.Context, c *CandidateContext) error
	// GetByJobID returns the context for a job, or ErrNotFound.
	GetByJobID(ctx context.Context, jobID uuid.UUID) (*CandidateContext, error)
}

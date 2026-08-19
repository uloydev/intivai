package domain

import (
	"context"

	"github.com/google/uuid"
)

type CandidateRepository interface {
	Create(ctx context.Context, c *Candidate) error
	GetByID(ctx context.Context, id uuid.UUID) (*Candidate, error)
	// ListByIDs — batched candidate fetch for list enrichment (kills N+1).
	ListByIDs(ctx context.Context, orgID uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]*Candidate, error)
	GetByReviewToken(ctx context.Context, token string) (*Candidate, error)
	// ConfirmReview — atomically confirm the extracted profile (cross-org,
	// SECURITY DEFINER); returns the candidate's org + id (uuid.Nil when the
	// token is invalid or the candidate is no longer pending review).
	ConfirmReview(ctx context.Context, token string, structured []byte) (orgID, candidateID uuid.UUID, err error)
	List(ctx context.Context, orgID uuid.UUID) ([]*Candidate, error)
	Update(ctx context.Context, candidate *Candidate) error
	Delete(ctx context.Context, id uuid.UUID) error
}

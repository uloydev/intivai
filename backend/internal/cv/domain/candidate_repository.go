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
	List(ctx context.Context, orgID uuid.UUID) ([]*Candidate, error)
	Update(ctx context.Context, candidate *Candidate) error
	Delete(ctx context.Context, id uuid.UUID) error
}

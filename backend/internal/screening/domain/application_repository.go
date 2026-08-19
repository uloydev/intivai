package domain

import (
	"context"

	"github.com/google/uuid"
)

type ApplicationRepository interface {
	Create(ctx context.Context, app *Application) error
	GetByID(ctx context.Context, id uuid.UUID) (*Application, error)
	GetByCandidateJob(ctx context.Context, orgID, candidateID, jobID uuid.UUID) (*Application, error)
	List(ctx context.Context, orgID, jobID uuid.UUID) ([]*Application, error)
	ByCandidate(ctx context.Context, orgID, candidateID uuid.UUID) ([]*Application, error)
	Update(ctx context.Context, app *Application) error
	// UpdateDecision persists the recruiter lifecycle stage + hiring notes
	// (column-scoped — does not clobber score columns).
	UpdateDecision(ctx context.Context, orgID, id uuid.UUID, stage *Stage, notes *string) error
	// ListByIDs — batched application fetch (kills N+1 list enrichment).
	ListByIDs(ctx context.Context, orgID uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]*Application, error)
	// ApplyWithDedupe — public-apply flow: advisory-locked find-or-create of
	// the candidate by (org, email), then an idempotent application insert.
	// Returns the candidate id and whether the candidate row is new.
	ApplyWithDedupe(ctx context.Context, orgID, jobID uuid.UUID, name, email string) (candidateID uuid.UUID, isNew bool, err error)
}

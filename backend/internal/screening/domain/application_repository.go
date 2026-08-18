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
}

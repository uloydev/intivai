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
	Update(ctx context.Context, app *Application) error
}

package domain

import (
	"context"

	"github.com/google/uuid"
)

type JobRepository interface {
	Create(ctx context.Context, job *Job) error
	GetByID(ctx context.Context, id uuid.UUID) (*Job, error)
	List(ctx context.Context, orgID uuid.UUID) ([]*Job, error)
	ListActive(ctx context.Context, orgID uuid.UUID) ([]*Job, error)
	Update(ctx context.Context, job *Job) error
}

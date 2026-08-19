package domain

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
)

type JobRepository interface {
	Create(ctx context.Context, job *Job) error
	GetByID(ctx context.Context, id uuid.UUID) (*Job, error)
	// ListByIDs — batched job fetch for list enrichment (kills N+1).
	ListByIDs(ctx context.Context, orgID uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]*Job, error)
	List(ctx context.Context, orgID uuid.UUID) ([]*Job, error)
	ListActive(ctx context.Context, orgID uuid.UUID) ([]*Job, error)
	Update(ctx context.Context, job *Job) error
	// UpdateRubric — column-scoped rubric write (never clobbers concurrent
	// recruiter edits via the full-row Update).
	UpdateRubric(ctx context.Context, id uuid.UUID, rubric json.RawMessage) error
}

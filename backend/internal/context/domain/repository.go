package domain

import (
	"context"

	"github.com/google/uuid"
)

type ContextRepository interface {
	CreateContext(ctx context.Context, cc *CompanyContext) error
	GetContextByID(ctx context.Context, id uuid.UUID) (*CompanyContext, error)
	GetContextByHash(ctx context.Context, orgID uuid.UUID, hash string) (*CompanyContext, error)
	ListContexts(ctx context.Context, orgID uuid.UUID) ([]*CompanyContext, error)
	SetPrompt(ctx context.Context, p *TenantPrompt) error
	GetLatestPrompt(ctx context.Context, orgID uuid.UUID) (*TenantPrompt, error)
}

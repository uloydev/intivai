package domain

import (
	"context"

	"github.com/google/uuid"
)

// Port — defined in the domain layer; driven adapters implement it.
type IAMRepository interface {
	CreateOrg(ctx context.Context, org *Org) error
	GetOrg(ctx context.Context, id uuid.UUID) (*Org, error)
	GetOrgBySlug(ctx context.Context, slug string) (*Org, error)
	CreateUser(ctx context.Context, user *User) error
	GetUserByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetUserByEmail(ctx context.Context, orgID uuid.UUID, email string) (*User, error)
	ListUsers(ctx context.Context, orgID uuid.UUID) ([]*User, error)
	// FindLoginIdentity is the pre-auth lookup used by login. It must work
	// WITHOUT a tenant context (security-definer function in Postgres).
	FindLoginIdentity(ctx context.Context, orgSlug, email string) (*LoginIdentity, error)
}

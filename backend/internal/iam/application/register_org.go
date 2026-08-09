package application

import (
	"context"
	"strings"

	"github.com/intivai/backend/internal/iam/domain"
	"github.com/intivai/backend/internal/shared/errors"
)

// RegisterOrg creates a tenant org + its first admin user.
// Runs in one transaction: the tenant (new org ID) is set BEFORE the first
// insert so the orgs RLS policy (USING id = app.org_id) passes for INSERT.
type RegisterOrg struct {
	repo   domain.IAMRepository
	hasher PasswordHasher
	tx     TxManager
}

func NewRegisterOrg(repo domain.IAMRepository, hasher PasswordHasher, tx TxManager) *RegisterOrg {
	return &RegisterOrg{repo: repo, hasher: hasher, tx: tx}
}

func (uc *RegisterOrg) Execute(ctx context.Context, cmd RegisterOrgCommand) (*RegisterOrgResult, error) {
	cmd.Name = strings.TrimSpace(cmd.Name)
	cmd.Slug = strings.ToLower(strings.TrimSpace(cmd.Slug))
	cmd.AdminEmail = strings.ToLower(strings.TrimSpace(cmd.AdminEmail))
	if len(cmd.AdminPassword) < 8 {
		return nil, errors.NewDomainError("WEAK_PASSWORD", "password must be at least 8 characters")
	}

	org, err := domain.NewOrg(cmd.Name, cmd.Slug)
	if err != nil {
		return nil, err
	}
	hash, err := uc.hasher.Hash(cmd.AdminPassword)
	if err != nil {
		return nil, err
	}
	admin, err := domain.NewUser(org.ID, cmd.AdminEmail, domain.RoleAdmin, hash, "password")
	if err != nil {
		return nil, err
	}

	err = uc.tx.RunInTx(ctx, &org.ID, func(tctx context.Context) error {
		if err := uc.repo.CreateOrg(tctx, org); err != nil {
			return err
		}
		return uc.repo.CreateUser(tctx, admin)
	})
	switch err {
	case nil:
		// ok
	case domain.ErrDuplicateSlug:
		return nil, errors.NewDomainError("ORG_SLUG_TAKEN", "org slug already taken")
	case domain.ErrDuplicateEmail:
		return nil, errors.NewDomainError("EMAIL_TAKEN", "email already registered in this org")
	default:
		return nil, err
	}

	return &RegisterOrgResult{OrgID: org.ID, UserID: admin.ID, Slug: org.Slug, Plan: org.Plan}, nil
}

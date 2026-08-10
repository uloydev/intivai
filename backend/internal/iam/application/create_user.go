package application

import (
	"context"
	"strings"

	iamdomain "github.com/intivai/backend/internal/iam/domain"
	"github.com/intivai/backend/internal/shared/errors"
)

// CreateUser adds a tenant member (invite flow: temp password, email delivery post-MVP).
type CreateUser struct {
	repo   iamdomain.IAMRepository
	hasher PasswordHasher
}

func NewCreateUser(repo iamdomain.IAMRepository, hasher PasswordHasher) *CreateUser {
	return &CreateUser{repo: repo, hasher: hasher}
}

// roleRank — creation authority order. Higher rank may create at-or-below.
func roleRank(r iamdomain.Role) int {
	switch r {
	case iamdomain.RoleAdmin:
		return 3
	case iamdomain.RoleRecruiter:
		return 2
	case iamdomain.RoleInterviewer:
		return 1
	default: // member
		return 0
	}
}

func canCreateRole(actor AuthContext, target iamdomain.Role) bool {
	if actor.Role == string(iamdomain.RoleAdmin) {
		return true
	}
	return roleRank(iamdomain.Role(actor.Role)) > roleRank(target)
}

func (uc *CreateUser) Execute(ctx context.Context, actor AuthContext, cmd CreateUserCommand) (*CreateUserResult, error) {
	if err := Authorize(actor, iamdomain.RoleAdmin, iamdomain.RoleRecruiter); err != nil {
		return nil, err
	}
	if len(cmd.Password) < 8 {
		return nil, errors.NewDomainError("WEAK_PASSWORD", "password must be at least 8 characters")
	}

	role := iamdomain.Role(cmd.Role)
	if cmd.Role == "" {
		role = iamdomain.RoleMember
	}
	if !role.Valid() {
		return nil, errors.NewDomainError("INVALID_ROLE", "invalid role: "+cmd.Role)
	}
	// No privilege escalation: a user may only create roles at or below
	// their own rank (admin > recruiter > interviewer/member).
	if !canCreateRole(actor, role) {
		return nil, errors.NewDomainError("FORBIDDEN", "cannot create a user with role "+string(role))
	}

	hash, err := uc.hasher.Hash(cmd.Password)
	if err != nil {
		return nil, err
	}
	user, err := iamdomain.NewUser(cmd.OrgID, strings.ToLower(strings.TrimSpace(cmd.Email)), role, hash, "password")
	if err != nil {
		return nil, err
	}
	if err := uc.repo.CreateUser(ctx, user); err != nil {
		if err == iamdomain.ErrDuplicateEmail {
			return nil, errors.NewDomainError("EMAIL_TAKEN", "email already registered in this org")
		}
		return nil, err
	}
	return &CreateUserResult{UserID: user.ID, Role: string(user.Role)}, nil
}

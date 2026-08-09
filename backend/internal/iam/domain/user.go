package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/intivai/backend/internal/shared/domain"
	"github.com/intivai/backend/internal/shared/errors"
)

type Role string

const (
	RoleAdmin       Role = "admin"
	RoleRecruiter   Role = "recruiter"
	RoleInterviewer Role = "interviewer"
	RoleMember      Role = "member"
)

func (r Role) Valid() bool {
	switch r {
	case RoleAdmin, RoleRecruiter, RoleInterviewer, RoleMember:
		return true
	}
	return false
}

// User is a tenant user. Unique per org: same email allowed in different orgs.
type User struct {
	domain.Entity
	OrgID        uuid.UUID
	Email        string
	Role         Role
	PasswordHash string // NULL if OAuth-only
	AuthProvider string // password | google
}

func NewUser(orgID uuid.UUID, email string, role Role, passwordHash, authProvider string) (*User, error) {
	if email == "" {
		return nil, errors.NewDomainError("USER_EMAIL_REQUIRED", "email is required")
	}
	if !role.Valid() {
		return nil, errors.NewDomainError("INVALID_ROLE", "invalid role: "+string(role))
	}
	if authProvider == "" {
		authProvider = "password"
	}
	return &User{
		Entity:       domain.Entity{ID: domain.NewID(), CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()},
		OrgID:        orgID,
		Email:        email,
		Role:         role,
		PasswordHash: passwordHash,
		AuthProvider: authProvider,
	}, nil
}

func (u *User) HasRole(roles ...Role) bool {
	for _, r := range roles {
		if u.Role == r {
			return true
		}
	}
	return false
}

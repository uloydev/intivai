package application

import (
	"time"

	"github.com/google/uuid"
)

// PasswordHasher — driven port, implemented by infrastructure/auth (bcrypt).
type PasswordHasher interface {
	Hash(plain string) (string, error)
	Verify(hash, plain string) bool
}

// TokenProvider — driven port, implemented by infrastructure/auth (JWT).
type TokenProvider interface {
	Issue(subject uuid.UUID, orgID uuid.UUID, role, tokenType string, ttl time.Duration, extra map[string]any) (string, error)
	Parse(token string) (*Claims, error)
}

type Claims struct {
	Subject uuid.UUID
	OrgID   uuid.UUID
	Role    string
	Type    string // "auth" | "ws_ticket"
	Extra   map[string]any
}

// AuthContext carries the authenticated identity through handlers.
type AuthContext struct {
	UserID uuid.UUID `json:"user_id"`
	OrgID  uuid.UUID `json:"org_id"`
	Role   string    `json:"role"`
}

// --- DTOs ---

type RegisterOrgCommand struct {
	Name          string
	Slug          string
	AdminEmail    string
	AdminPassword string
}

type RegisterOrgResult struct {
	OrgID  uuid.UUID `json:"org_id"`
	UserID uuid.UUID `json:"user_id"`
	Slug   string    `json:"slug"`
	Plan   string    `json:"plan"`
}

type AuthenticateCommand struct {
	OrgSlug  string
	Email    string
	Password string
}

type AuthenticateResult struct {
	Token     string      `json:"token"`
	ExpiresAt time.Time   `json:"expires_at"`
	User      AuthContext `json:"user"`
}

type CreateUserCommand struct {
	OrgID    uuid.UUID
	Email    string
	Role     string
	Password string // temporary; hashed before persist
}

type CreateUserResult struct {
	UserID uuid.UUID `json:"user_id"`
	Role   string    `json:"role"`
}

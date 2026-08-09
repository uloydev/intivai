package domain

import (
	"time"

	"github.com/google/uuid"
)

// LoginIdentity is the pre-auth lookup result (security-definer path).
// PasswordHash is nil for OAuth-only users.
type LoginIdentity struct {
	OrgID        uuid.UUID
	UserID       uuid.UUID
	Email        string
	PasswordHash *string
	Role         Role
	AuthProvider string
	CreatedAt    time.Time
}

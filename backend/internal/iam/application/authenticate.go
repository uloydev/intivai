package application

import (
	"context"
	"time"

	"github.com/google/uuid"
	iamdomain "github.com/intivai/backend/internal/iam/domain"
	"github.com/intivai/backend/internal/shared/errors"
)

// Token types — auth tokens and short-lived WS tickets are distinguishable.
const (
	TokenTypeAuth     = "auth"
	TokenTypeWSTicket = "ws_ticket"
)

// Authenticate validates credentials via the pre-auth security-definer lookup
// and issues a JWT.
type Authenticate struct {
	repo   iamdomain.IAMRepository
	hasher PasswordHasher
	tokens TokenProvider
	ttl    time.Duration
}

func NewAuthenticate(repo iamdomain.IAMRepository, hasher PasswordHasher, tokens TokenProvider, ttl time.Duration) *Authenticate {
	return &Authenticate{repo: repo, hasher: hasher, tokens: tokens, ttl: ttl}
}

func (uc *Authenticate) Execute(ctx context.Context, cmd AuthenticateCommand) (*AuthenticateResult, error) {
	id, err := uc.repo.FindLoginIdentity(ctx, cmd.OrgSlug, cmd.Email)
	if err != nil {
		return nil, errors.NewDomainError("AUTH_FAILED", "invalid credentials")
	}
	if id.PasswordHash == nil || !uc.hasher.Verify(*id.PasswordHash, cmd.Password) {
		return nil, errors.NewDomainError("AUTH_FAILED", "invalid credentials")
	}

	token, err := uc.tokens.Issue(id.UserID, id.OrgID, string(id.Role), TokenTypeAuth, uc.ttl, nil)
	if err != nil {
		return nil, err
	}

	return &AuthenticateResult{
		Token:     token,
		ExpiresAt: time.Now().UTC().Add(uc.ttl),
		User: AuthContext{
			UserID: id.UserID,
			OrgID:  id.OrgID,
			Role:   string(id.Role),
		},
	}, nil
}

// Authorize checks role-based access. Use inside use cases that need RBAC.
func Authorize(actor AuthContext, required ...iamdomain.Role) error {
	for _, r := range required {
		if iamdomain.Role(actor.Role) == r {
			return nil
		}
	}
	return errors.NewDomainError("FORBIDDEN", "insufficient role")
}

// UserIDFromString helper for use cases that receive raw UUIDs from handlers.
func UserIDFromString(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}

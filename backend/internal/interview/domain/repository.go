package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type InterviewRepository interface {
	Create(ctx context.Context, iv *Interview) error
	GetByID(ctx context.Context, id uuid.UUID) (*Interview, error)
	Update(ctx context.Context, iv *Interview) error
	// SaveEvaluation persists the post-interview report (evaluation JSONB).
	SaveEvaluation(ctx context.Context, id uuid.UUID, report []byte) error
	// ByApplication lists interviews for an application (recruiter report).
	ByApplication(ctx context.Context, applicationID uuid.UUID) ([]*Interview, error)
}

// TokenStatus — result of validating an invitation token (definer function).
type TokenStatus string

const (
	TokenValid    TokenStatus = "valid"
	TokenExpired  TokenStatus = "expired"
	TokenUsed     TokenStatus = "used"
	TokenRevoked  TokenStatus = "revoked"
	TokenNotFound TokenStatus = "not_found"
)

// InvitationToken — 32-char high-entropy credential (Research §3).
type InvitationToken struct {
	ID          uuid.UUID
	OrgID       uuid.UUID
	InterviewID uuid.UUID
	Token       string
	ExpiresAt   time.Time
	UsedAt      *time.Time
	RevokedAt   *time.Time
	CreatedAt   time.Time
}

type TokenRepository interface {
	Create(ctx context.Context, t *InvitationToken) error
	// Validate is pre-auth: security-definer, no tenant context required.
	Validate(ctx context.Context, token string) (*InvitationToken, TokenStatus)
	MarkUsed(ctx context.Context, token string) error
}

// QuestionBank — generated questions persisted for reuse + audit.
type QuestionBank interface {
	Create(ctx context.Context, orgID uuid.UUID, q Question) error
	ListByOrg(ctx context.Context, orgID uuid.UUID) ([]Question, error)
}

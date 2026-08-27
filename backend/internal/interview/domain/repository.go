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
	// SetConsent records GDPR consent (consent_given), idempotent.
	SetConsent(ctx context.Context, id uuid.UUID) error
	// RecordProctoringEvent persists an integrity telemetry event on the interview.
	RecordProctoringEvent(ctx context.Context, id uuid.UUID, event ProctoringEvent) error
	// Touch refreshes the activity marker (updated_at/expires_at) WITHOUT
	// rewriting the transcript — full-row Update() racing answer commits
	// would clobber persisted answers with a stale aggregate.
	Touch(ctx context.Context, id uuid.UUID) error
	// RecordCodingSession appends a coding snapshot without rewriting the
	// transcript (same lost-update concern as Touch).
	RecordCodingSession(ctx context.Context, id uuid.UUID, session CodingSession) error
	// AppendQAPairWithinLimit appends a candidate Q&A pair (B4) without
	// rewriting the transcript — candidate questions must never touch the
	// scored answers. The per-interview cap (limit) AND the active/expiry
	// predicate are enforced AT THE SQL LAYER (single UPDATE) so concurrent
	// frames cannot race past them (J6). Outcomes:
	//   - (true, nil): pair appended.
	//   - (false, nil): cap reached, nothing written.
	//   - (false, ErrQAActive): interview not in_progress or expired.
	//   - (false, domain.ErrNotFound): interview row missing.
	AppendQAPairWithinLimit(ctx context.Context, id uuid.UUID, pair QAPair, limit int) (bool, error)
	// ExpireIfDue flips an overdue in_progress interview to expired —
	// column-scoped so the QA path can persist honest expiry state without a
	// full-row rewrite.
	ExpireIfDue(ctx context.Context, id uuid.UUID) error
	// SetHumanRequested marks the interview as having a candidate-requested
	// human interviewer (idempotent).
	SetHumanRequested(ctx context.Context, id uuid.UUID, requested bool) error
	// ListByOrg lists the org's interviews (RLS-scoped via the applications
	// join), newest first.
	ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*Interview, error)
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

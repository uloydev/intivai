package persistence

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	ivdomain "github.com/intivai/backend/internal/interview/domain"
	"github.com/intivai/backend/pkg/db"
	"gorm.io/gorm"
)

type PostgresTokenRepo struct {
	pool *gorm.DB
}

func NewPostgresTokenRepo(pool *gorm.DB) *PostgresTokenRepo {
	return &PostgresTokenRepo{pool: pool}
}

func (r *PostgresTokenRepo) q(ctx context.Context) (*gorm.DB, error) {
	tx, ok := db.TxFrom(ctx)
	if !ok {
		return nil, db.ErrNoTx
	}
	return tx, nil
}

func (r *PostgresTokenRepo) Create(ctx context.Context, t *ivdomain.InvitationToken) error {
	q, err := r.q(ctx)
	if err != nil {
		return err
	}
	return q.WithContext(ctx).Exec(
		`INSERT INTO interview_tokens (id, org_id, interview_id, token, expires_at, created_at)
		 VALUES ($1, $2, $3, $4, $5, NOW())`,
		t.ID, t.OrgID, t.InterviewID, t.Token, t.ExpiresAt).Error
}

// Validate is pre-auth: uses the security-definer validate_interview_token —
// works WITHOUT a tenant context (candidates never touch tables directly).
func (r *PostgresTokenRepo) Validate(ctx context.Context, token string) (*ivdomain.InvitationToken, ivdomain.TokenStatus) {
	row := r.pool.WithContext(ctx).Raw(
		`SELECT token_id, interview_id, org_id, status FROM validate_interview_token($1)`, token).Row()
	var (
		tokenID, interviewID, orgID uuid.UUID
		status                      string
	)
	if err := row.Scan(&tokenID, &interviewID, &orgID, &status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ivdomain.TokenNotFound
		}
		return nil, ivdomain.TokenNotFound
	}
	return &ivdomain.InvitationToken{ID: tokenID, InterviewID: interviewID, OrgID: orgID}, ivdomain.TokenStatus(status)
}

// MarkUsed sets used_at on FIRST start. Runs in a tenant tx (RLS table).
func (r *PostgresTokenRepo) MarkUsed(ctx context.Context, token string) error {
	q, err := r.q(ctx)
	if err != nil {
		return err
	}
	return q.WithContext(ctx).Exec(
		`UPDATE interview_tokens SET used_at = NOW() WHERE token = $1 AND used_at IS NULL`, token).Error
}

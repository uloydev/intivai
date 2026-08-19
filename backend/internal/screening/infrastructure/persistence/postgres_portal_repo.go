package persistence

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	scrdomain "github.com/intivai/backend/internal/screening/domain"
	"gorm.io/gorm"
)

// PostgresCandidatePortalRepo — candidate portal persistence. OTP rows are
// not tenant-scoped (auth happens before any tenant context exists), so these
// queries run outside tenant transactions by design.
type PostgresCandidatePortalRepo struct {
	pool *gorm.DB
}

func NewPostgresCandidatePortalRepo(pool *gorm.DB) *PostgresCandidatePortalRepo {
	return &PostgresCandidatePortalRepo{pool: pool}
}

func (r *PostgresCandidatePortalRepo) CreateOTP(ctx context.Context, email, codeHash, token string, expiresAt time.Time) error {
	return r.pool.WithContext(ctx).Exec(
		`INSERT INTO candidate_otps (id, email, code_hash, token, expires_at, created_at)
		 VALUES (?, ?, ?, ?, ?, NOW())`,
		uuid.New(), email, codeHash, token, expiresAt,
	).Error
}

func (r *PostgresCandidatePortalRepo) LastRequestAt(ctx context.Context, email string) (*time.Time, error) {
	var last sql.NullTime
	err := r.pool.WithContext(ctx).Raw(
		`SELECT MAX(created_at) FROM candidate_otps WHERE LOWER(email) = ? AND used_at IS NULL`, email,
	).Row().Scan(&last)
	if err != nil {
		return nil, err
	}
	if !last.Valid {
		return nil, nil
	}
	return &last.Time, nil
}

func (r *PostgresCandidatePortalRepo) OTPCountSince(ctx context.Context, email string, since time.Time) (int, error) {
	var n int
	err := r.pool.WithContext(ctx).Raw(
		`SELECT COUNT(*) FROM candidate_otps WHERE LOWER(email) = ? AND created_at > ?`, email, since,
	).Row().Scan(&n)
	return n, err
}

func scanOTP(row *sql.Row) (*scrdomain.CandidateOTP, error) {
	var o scrdomain.CandidateOTP
	err := row.Scan(&o.ID, &o.Email, &o.CodeHash, &o.Token, &o.Attempts, &o.ExpiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (r *PostgresCandidatePortalRepo) FindValidByToken(ctx context.Context, token string) (*scrdomain.CandidateOTP, error) {
	row := r.pool.WithContext(ctx).Raw(
		`SELECT id, email, code_hash, token, attempts, expires_at FROM candidate_otps
		 WHERE token = ? AND used_at IS NULL AND expires_at > NOW()
		 ORDER BY created_at DESC LIMIT 1`, token).Row()
	return scanOTP(row)
}

func (r *PostgresCandidatePortalRepo) FindValidByCodeHash(ctx context.Context, email, codeHash string) (*scrdomain.CandidateOTP, error) {
	row := r.pool.WithContext(ctx).Raw(
		`SELECT id, email, code_hash, token, attempts, expires_at FROM candidate_otps
		 WHERE LOWER(email) = ? AND code_hash = ? AND used_at IS NULL AND expires_at > NOW()
		 ORDER BY created_at DESC LIMIT 1`, email, codeHash).Row()
	return scanOTP(row)
}

func (r *PostgresCandidatePortalRepo) IncrementAttempts(ctx context.Context, email string) error {
	return r.pool.WithContext(ctx).Exec(
		`UPDATE candidate_otps SET attempts = attempts + 1
		 WHERE LOWER(email) = ? AND used_at IS NULL AND expires_at > NOW()`, email).Error
}

func (r *PostgresCandidatePortalRepo) Consume(ctx context.Context, id uuid.UUID) (bool, error) {
	res := r.pool.WithContext(ctx).Exec(
		`UPDATE candidate_otps SET used_at = NOW() WHERE id = ? AND used_at IS NULL`, id)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

func (r *PostgresCandidatePortalRepo) PurgeExpired(ctx context.Context, email string) error {
	return r.pool.WithContext(ctx).Exec(
		`DELETE FROM candidate_otps WHERE LOWER(email) = ? AND (expires_at < NOW() OR used_at IS NOT NULL)`, email).Error
}

func (r *PostgresCandidatePortalRepo) EraseCandidate(ctx context.Context, email string) error {
	// SECURITY DEFINER function (intivai_rls_bypass owner) — the app user is
	// RLS-bound to one org and cannot delete across tenants.
	return r.pool.WithContext(ctx).Exec(`SELECT candidate_erase(?)`, email).Error
}

// ListApplications — the candidate's applications across orgs via the
// SECURITY DEFINER lookup function.
func (r *PostgresCandidatePortalRepo) ListApplications(ctx context.Context, email string) ([]*scrdomain.CandidateApplicationView, error) {
	rows, err := r.pool.WithContext(ctx).Raw(
		`SELECT application_id, org_id, org_name, org_slug, job_id, job_title, job_location, job_employment_type,
		        candidate_id, candidate_name, candidate_email, cv_score, passed_screening, application_status,
		        applied_at, interview_id, interview_status, interview_type, invitation_token, overall_score, recommendation
		 FROM candidate_applications_lookup(?)`, email).Rows()
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	out := []*scrdomain.CandidateApplicationView{}
	for rows.Next() {
		var a scrdomain.CandidateApplicationView
		var appliedAt time.Time
		if err := rows.Scan(
			&a.ApplicationID, &a.OrgID, &a.OrgName, &a.OrgSlug, &a.JobID, &a.JobTitle, &a.JobLocation, &a.JobEmploymentType,
			&a.CandidateID, &a.CandidateName, &a.CandidateEmail, &a.CVScore, &a.PassedScreening, &a.ApplicationStatus,
			&appliedAt, &a.InterviewID, &a.InterviewStatus, &a.InterviewType, &a.InvitationToken, &a.OverallScore, &a.Recommendation,
		); err != nil {
			return nil, err
		}
		a.AppliedAt = appliedAt.Format(time.RFC3339)
		out = append(out, &a)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

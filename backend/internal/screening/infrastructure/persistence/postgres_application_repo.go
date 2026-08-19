package persistence

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	scrdomain "github.com/intivai/backend/internal/screening/domain"
	"github.com/intivai/backend/pkg/db"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

type PostgresApplicationRepo struct {
	pool *gorm.DB
}

func NewPostgresApplicationRepo(pool *gorm.DB) *PostgresApplicationRepo {
	return &PostgresApplicationRepo{pool: pool}
}

func (r *PostgresApplicationRepo) q(ctx context.Context) (*gorm.DB, error) {
	tx, ok := db.TxFrom(ctx)
	if !ok {
		return nil, db.ErrNoTx
	}
	return tx, nil
}

func (r *PostgresApplicationRepo) Create(ctx context.Context, app *scrdomain.Application) error {
	q, err := r.q(ctx)
	if err != nil {
		return err
	}
	err = q.WithContext(ctx).Exec(
		`INSERT INTO applications (id, org_id, candidate_id, job_id, status, stage, created_at)
		 VALUES ($1, $2, $3, $4, $5, 'applied', $6)`,
		app.ID, app.OrgID, app.CandidateID, app.JobID, app.Status, app.CreatedAt).Error
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return scrdomain.ErrExists
	}
	return err
}

func (r *PostgresApplicationRepo) GetByID(ctx context.Context, id uuid.UUID) (*scrdomain.Application, error) {
	q, err := r.q(ctx)
	if err != nil {
		return nil, err
	}
	row := q.Raw(
		`SELECT id, org_id, candidate_id, job_id, cv_score, score_breakdown, passed_screening, status, stage, recruiter_notes, created_at
		 FROM applications WHERE id = $1`, id).Row()
	return scanApplication(row)
}

func (r *PostgresApplicationRepo) GetByCandidateJob(ctx context.Context, orgID, candidateID, jobID uuid.UUID) (*scrdomain.Application, error) {
	q, err := r.q(ctx)
	if err != nil {
		return nil, err
	}
	row := q.Raw(
		`SELECT id, org_id, candidate_id, job_id, cv_score, score_breakdown, passed_screening, status, stage, recruiter_notes, created_at
		 FROM applications WHERE org_id = $1 AND candidate_id = $2 AND job_id = $3`,
		orgID, candidateID, jobID).Row()
	return scanApplication(row)
}

func (r *PostgresApplicationRepo) List(ctx context.Context, orgID, jobID uuid.UUID) ([]*scrdomain.Application, error) {
	q, err := r.q(ctx)
	if err != nil {
		return nil, err
	}
	sql := `SELECT id, org_id, candidate_id, job_id, cv_score, score_breakdown, passed_screening, status, stage, recruiter_notes, created_at
		FROM applications WHERE org_id = $1`
	args := []any{orgID}
	if jobID != uuid.Nil {
		sql += ` AND job_id = $2`
		args = append(args, jobID)
	}
	sql += ` ORDER BY created_at DESC`
	rows, err := q.Raw(sql, args...).Rows()
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []*scrdomain.Application{}
	for rows.Next() {
		a, err := scanApplication(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// ByCandidate returns all applications for one candidate (report view).
func (r *PostgresApplicationRepo) ByCandidate(ctx context.Context, orgID, candidateID uuid.UUID) ([]*scrdomain.Application, error) {
	q, err := r.q(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := q.Raw(
		`SELECT id, org_id, candidate_id, job_id, cv_score, score_breakdown, passed_screening, status, stage, recruiter_notes, created_at
		 FROM applications WHERE org_id = $1 AND candidate_id = $2 ORDER BY created_at DESC`,
		orgID, candidateID).Rows()
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []*scrdomain.Application{}
	for rows.Next() {
		a, err := scanApplication(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *PostgresApplicationRepo) Update(ctx context.Context, app *scrdomain.Application) error {
	q, err := r.q(ctx)
	if err != nil {
		return err
	}
	return q.WithContext(ctx).Exec(
		`UPDATE applications SET cv_score = $1, score_breakdown = $2, passed_screening = $3,
		 status = $4, updated_at = NOW() WHERE id = $5`,
		app.CVScore, app.ScoreBreakdown, app.PassedScreening, app.Status, app.ID).Error
}

// UpdateDecision persists stage + recruiter_notes only (PATCH semantics —
// nil = keep). org_id is belt-and-braces on top of RLS.
func (r *PostgresApplicationRepo) UpdateDecision(ctx context.Context, orgID, id uuid.UUID, stage *scrdomain.Stage, notes *string) error {
	q, err := r.q(ctx)
	if err != nil {
		return err
	}
	var stageVal *string
	if stage != nil {
		v := string(*stage)
		stageVal = &v
	}
	res := q.WithContext(ctx).Exec(
		`UPDATE applications SET
		   stage = COALESCE($1, stage),
		   recruiter_notes = COALESCE($2, recruiter_notes),
		   updated_at = NOW()
		 WHERE id = $3 AND org_id = $4`,
		stageVal, notes, id, orgID)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return scrdomain.ErrNotFound
	}
	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanApplication(row rowScanner) (*scrdomain.Application, error) {
	var (
		a         scrdomain.Application
		score     *float64
		breakdown []byte
		passed    *bool
	)
	err := row.Scan(&a.ID, &a.OrgID, &a.CandidateID, &a.JobID, &score, &breakdown, &passed, &a.Status, &a.Stage, &a.RecruiterNotes, &a.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, scrdomain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	a.CVScore = score
	if len(breakdown) > 0 {
		a.ScoreBreakdown = breakdown
	}
	a.PassedScreening = passed
	return &a, nil
}

package persistence

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	cvdomain "github.com/intivai/backend/internal/cv/domain"
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

func (r *PostgresApplicationRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := db.TxFrom(ctx)
	if !ok {
		return nil, db.ErrNoTx
	}
	return tx, nil
}

func (r *PostgresApplicationRepo) Create(ctx context.Context, app *scrdomain.Application) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	err = tx.WithContext(ctx).Exec(
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
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	row := tx.Raw(
		`SELECT id, org_id, candidate_id, job_id, cv_score, score_breakdown, passed_screening, status, stage, recruiter_notes, created_at
		 FROM applications WHERE id = $1`, id).Row()
	return scanApplication(row)
}

func (r *PostgresApplicationRepo) GetByCandidateJob(ctx context.Context, orgID, candidateID, jobID uuid.UUID) (*scrdomain.Application, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	row := tx.Raw(
		`SELECT id, org_id, candidate_id, job_id, cv_score, score_breakdown, passed_screening, status, stage, recruiter_notes, created_at
		 FROM applications WHERE org_id = $1 AND candidate_id = $2 AND job_id = $3`,
		orgID, candidateID, jobID).Row()
	return scanApplication(row)
}

func (r *PostgresApplicationRepo) List(ctx context.Context, orgID, jobID uuid.UUID) ([]*scrdomain.Application, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	sql := `SELECT id, org_id, candidate_id, job_id, cv_score, score_breakdown, passed_screening, status, stage, recruiter_notes, created_at,
		(SELECT (evaluation->>'overall')::float8 FROM interviews
		 WHERE application_id = applications.id AND evaluation IS NOT NULL
		 ORDER BY created_at DESC LIMIT 1) AS interview_score
		FROM applications WHERE org_id = $1`
	args := []any{orgID}
	if jobID != uuid.Nil {
		sql += ` AND job_id = $2`
		args = append(args, jobID)
	}
	sql += ` ORDER BY created_at DESC`
	rows, err := tx.Raw(sql, args...).Rows()
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []*scrdomain.Application{}
	for rows.Next() {
		a, err := scanApplicationWithScore(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// scanApplicationWithScore — scanApplication + the list-only interview_score
// column, scanned in ONE call (database/sql requires all columns per Scan).
func scanApplicationWithScore(rows *sql.Rows) (*scrdomain.Application, error) {
	var (
		a         scrdomain.Application
		score     *float64
		breakdown []byte
		passed    *bool
	)
	err := rows.Scan(&a.ID, &a.OrgID, &a.CandidateID, &a.JobID, &score, &breakdown, &passed, &a.Status, &a.Stage, &a.RecruiterNotes, &a.CreatedAt, &a.InterviewScore)
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

// ByCandidate returns all applications for one candidate (report view).
func (r *PostgresApplicationRepo) ByCandidate(ctx context.Context, orgID, candidateID uuid.UUID) ([]*scrdomain.Application, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := tx.Raw(
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
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	return tx.WithContext(ctx).Exec(
		`UPDATE applications SET cv_score = $1, score_breakdown = $2, passed_screening = $3,
		 status = $4, updated_at = NOW() WHERE id = $5`,
		app.CVScore, app.ScoreBreakdown, app.PassedScreening, app.Status, app.ID).Error
}

// UpdateDecision persists stage + recruiter_notes only (PATCH semantics —
// nil = keep). org_id is belt-and-braces on top of RLS.
func (r *PostgresApplicationRepo) UpdateDecision(ctx context.Context, orgID, id uuid.UUID, stage *scrdomain.Stage, notes *string) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	var stageVal *string
	if stage != nil {
		v := string(*stage)
		stageVal = &v
	}
	res := tx.WithContext(ctx).Exec(
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

// ApplyWithDedupe — public-apply flow (ADR-0001 stage start + dedupe):
// advisory lock per (org, email), reuse an existing candidate row, else
// create it (status 'parsing'), then insert the application idempotently.
// Returns the candidate id and whether the candidate row was newly created.
func (r *PostgresApplicationRepo) ApplyWithDedupe(ctx context.Context, orgID, jobID uuid.UUID, name, email string) (uuid.UUID, bool, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return uuid.Nil, false, err
	}
	email = strings.ToLower(email)
	lockKey := orgID.String() + ":" + email
	if err := tx.WithContext(ctx).Exec(`SELECT pg_advisory_xact_lock(hashtext(?))`, lockKey).Error; err != nil {
		return uuid.Nil, false, err
	}

	var existing uuid.UUID
	row := tx.WithContext(ctx).Raw(
		`SELECT id FROM candidates WHERE org_id = ? AND LOWER(email) = ? ORDER BY created_at LIMIT 1`,
		orgID, email).Row()
	switch err := row.Scan(&existing); {
	case err == nil:
		return existing, false, nil
	case !errors.Is(err, sql.ErrNoRows):
		return uuid.Nil, false, err
	}

	candID := uuid.New()
	cvPath := fmt.Sprintf("cvs/%s/%s.pdf", orgID, candID)
	if err := tx.WithContext(ctx).Exec(
		`INSERT INTO candidates (id, org_id, name, email, cv_path, status, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())`,
		candID, orgID, name, email, cvPath, cvdomain.StatusParsing).Error; err != nil {
		return uuid.Nil, false, err
	}
	if err := tx.WithContext(ctx).Exec(
		`INSERT INTO applications (id, org_id, candidate_id, job_id, status, stage, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, 'screening', 'applied', NOW(), NOW())
		 ON CONFLICT (candidate_id, job_id) DO UPDATE SET updated_at = NOW()`,
		uuid.New(), orgID, candID, jobID).Error; err != nil {
		return uuid.Nil, false, err
	}
	return candID, true, nil
}

// ListByIDs — batched application fetch (kills N+1 list enrichment).
func (r *PostgresApplicationRepo) ListByIDs(ctx context.Context, orgID uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]*scrdomain.Application, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return map[uuid.UUID]*scrdomain.Application{}, nil
	}
	rows, err := tx.WithContext(ctx).Raw(
		`SELECT id, org_id, candidate_id, job_id, cv_score, score_breakdown, passed_screening, status, stage, recruiter_notes, created_at
		 FROM applications WHERE org_id = $1 AND id = ANY($2)`, orgID, ids).Rows()
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := map[uuid.UUID]*scrdomain.Application{}
	for rows.Next() {
		a, err := scanApplication(rows)
		if err != nil {
			return nil, err
		}
		out[a.ID] = a
	}
	return out, rows.Err()
}

package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	jobdomain "github.com/intivai/backend/internal/job/domain"
	"github.com/intivai/backend/pkg/db"
	"gorm.io/gorm"
)

const jobColumns = `id, org_id, title, description, required_skills, min_experience, scoring_weights, min_score_to_proceed, status, created_at`

type PostgresJobRepo struct {
	pool *gorm.DB
}

func NewPostgresJobRepo(pool *gorm.DB) *PostgresJobRepo {
	return &PostgresJobRepo{pool: pool}
}

func (r *PostgresJobRepo) q(ctx context.Context) (*gorm.DB, error) {
	tx, ok := db.TxFrom(ctx)
	if !ok {
		return nil, db.ErrNoTx
	}
	return tx, nil
}

func (r *PostgresJobRepo) Create(ctx context.Context, job *jobdomain.Job) error {
	q, err := r.q(ctx)
	if err != nil {
		return err
	}
	return q.WithContext(ctx).Exec(
		`INSERT INTO jobs (id, org_id, title, description, required_skills, min_experience, scoring_weights, min_score_to_proceed, status, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		job.ID, job.OrgID, job.Title, job.Description, job.RequiredSkills, job.MinExperience,
		job.MarshalScoringWeights(), job.MinScoreToProceed, job.Status, job.CreatedAt).Error
}

func (r *PostgresJobRepo) GetByID(ctx context.Context, id uuid.UUID) (*jobdomain.Job, error) {
	q, err := r.q(ctx)
	if err != nil {
		return nil, err
	}
	row := q.Raw(
		`SELECT `+jobColumns+` FROM jobs WHERE id = $1`, id).Row()
	return scanJob(row)
}

func (r *PostgresJobRepo) List(ctx context.Context, orgID uuid.UUID) ([]*jobdomain.Job, error) {
	return r.list(ctx, orgID, "")
}

func (r *PostgresJobRepo) ListActive(ctx context.Context, orgID uuid.UUID) ([]*jobdomain.Job, error) {
	return r.list(ctx, orgID, ` AND status = 'active'`)
}

func (r *PostgresJobRepo) list(ctx context.Context, orgID uuid.UUID, extra string) ([]*jobdomain.Job, error) {
	q, err := r.q(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := q.Raw(
		`SELECT `+jobColumns+` FROM jobs WHERE org_id = $1`+extra+` ORDER BY created_at`, orgID).Rows()
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	jobs := []*jobdomain.Job{}
	for rows.Next() {
		j, err := scanJob(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}
	return jobs, rows.Err()
}

func (r *PostgresJobRepo) Update(ctx context.Context, job *jobdomain.Job) error {
	q, err := r.q(ctx)
	if err != nil {
		return err
	}
	return q.WithContext(ctx).Exec(
		`UPDATE jobs SET title = $1, description = $2, required_skills = $3, min_experience = $4,
		 scoring_weights = $5, min_score_to_proceed = $6, status = $7, updated_at = NOW() WHERE id = $8`,
		job.Title, job.Description, job.RequiredSkills, job.MinExperience,
		job.MarshalScoringWeights(), job.MinScoreToProceed, job.Status, job.ID).Error
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanJob(row rowScanner) (*jobdomain.Job, error) {
	var (
		j             jobdomain.Job
		skills        *[]byte
		weights       []byte
		minScore      *float64
		minExperience *int
	)
	err := row.Scan(&j.ID, &j.OrgID, &j.Title, &j.Description, &skills, &minExperience, &weights, &minScore, &j.Status, &j.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, jobdomain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if minExperience != nil {
		j.MinExperience = *minExperience
	}
	if skills != nil && len(*skills) > 0 && string(*skills) != "null" {
		_ = json.Unmarshal(*skills, &j.RequiredSkills)
	}
	j.MinScoreToProceed = minScore
	if len(weights) > 0 {
		_ = json.Unmarshal(weights, &j.ScoringWeights)
	}
	return &j, nil
}

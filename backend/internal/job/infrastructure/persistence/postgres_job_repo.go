package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	jobdomain "github.com/intivai/backend/internal/job/domain"
	"github.com/intivai/backend/pkg/db"
	"gorm.io/gorm"
)

const jobColumns = `id, org_id, title, description, location, employment_type, salary_min, salary_max, currency, required_skills, min_experience, responsibilities, requirements, nice_to_haves, benefits, scoring_weights, min_score_to_proceed, status, proctoring_mode, is_published, rubric, question_set_error, created_at, updated_at`

// publicJobColumns is the column list returned by the public job lookup
// functions (public_active_jobs_lookup / public_job_detail_lookup).
const publicJobColumns = `id, org_id, org_name, org_slug, title, description, location, employment_type,
		        salary_min, salary_max, currency, required_skills, min_experience,
		        responsibilities, requirements, nice_to_haves, benefits, status, proctoring_mode, is_published, rubric, created_at`

type PostgresJobRepo struct {
	pool *gorm.DB
}

func NewPostgresJobRepo(pool *gorm.DB) *PostgresJobRepo {
	return &PostgresJobRepo{pool: pool}
}

func (r *PostgresJobRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := db.TxFrom(ctx)
	if !ok {
		return nil, db.ErrNoTx
	}
	return tx, nil
}

func (r *PostgresJobRepo) Create(ctx context.Context, job *jobdomain.Job) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	reqSkills, err := marshalJSONB(job.RequiredSkills)
	if err != nil {
		return fmt.Errorf("encode required skills: %w", err)
	}
	resp, err := marshalJSONB(job.Responsibilities)
	if err != nil {
		return fmt.Errorf("encode responsibilities: %w", err)
	}
	reqs, err := marshalJSONB(job.Requirements)
	if err != nil {
		return fmt.Errorf("encode requirements: %w", err)
	}
	nice, err := marshalJSONB(job.NiceToHaves)
	if err != nil {
		return fmt.Errorf("encode nice-to-haves: %w", err)
	}
	ben, err := marshalJSONB(job.Benefits)
	if err != nil {
		return fmt.Errorf("encode benefits: %w", err)
	}
	weights, err := job.MarshalScoringWeights()
	if err != nil {
		return fmt.Errorf("encode scoring weights: %w", err)
	}

	return tx.WithContext(ctx).Exec(
		`INSERT INTO jobs (id, org_id, title, description, location, employment_type, salary_min, salary_max, currency,
		                   required_skills, min_experience, responsibilities, requirements, nice_to_haves, benefits,
		                   scoring_weights, min_score_to_proceed, status, proctoring_mode, is_published, rubric, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23)`,
		job.ID, job.OrgID, job.Title, job.Description, job.Location, job.EmploymentType, job.SalaryMin, job.SalaryMax, job.Currency,
		reqSkills, job.MinExperience, resp, reqs, nice, ben,
		weights, job.MinScoreToProceed, job.Status, job.ProctoringMode, job.IsPublished, job.Rubric, job.CreatedAt, job.CreatedAt).Error
}

func (r *PostgresJobRepo) GetByID(ctx context.Context, id uuid.UUID) (*jobdomain.Job, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	row := tx.Raw(
		`SELECT `+jobColumns+` FROM jobs WHERE id = $1`, id).Row()
	return scanJob(row)
}

// GetByIDForUpdate — locked read (D8): FOR UPDATE holds the row lock until
// the surrounding tenant transaction ends, serializing concurrent
// publish-sensitive updates against the same job row.
func (r *PostgresJobRepo) GetByIDForUpdate(ctx context.Context, id uuid.UUID) (*jobdomain.Job, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	row := tx.Raw(
		`SELECT `+jobColumns+` FROM jobs WHERE id = $1 FOR UPDATE`, id).Row()
	return scanJob(row)
}

func (r *PostgresJobRepo) List(ctx context.Context, orgID uuid.UUID) ([]*jobdomain.Job, error) {
	return r.list(ctx, orgID, "")
}

func (r *PostgresJobRepo) ListActive(ctx context.Context, orgID uuid.UUID) ([]*jobdomain.Job, error) {
	return r.list(ctx, orgID, ` AND status = 'active'`)
}

func (r *PostgresJobRepo) list(ctx context.Context, orgID uuid.UUID, extra string) ([]*jobdomain.Job, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := tx.Raw(
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
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	reqSkills, err := marshalJSONB(job.RequiredSkills)
	if err != nil {
		return fmt.Errorf("encode required skills: %w", err)
	}
	resp, err := marshalJSONB(job.Responsibilities)
	if err != nil {
		return fmt.Errorf("encode responsibilities: %w", err)
	}
	reqs, err := marshalJSONB(job.Requirements)
	if err != nil {
		return fmt.Errorf("encode requirements: %w", err)
	}
	nice, err := marshalJSONB(job.NiceToHaves)
	if err != nil {
		return fmt.Errorf("encode nice-to-haves: %w", err)
	}
	ben, err := marshalJSONB(job.Benefits)
	if err != nil {
		return fmt.Errorf("encode benefits: %w", err)
	}
	weights, err := job.MarshalScoringWeights()
	if err != nil {
		return fmt.Errorf("encode scoring weights: %w", err)
	}

	// RETURNING updated_at keeps the in-memory aggregate in sync with the row
	// version the DB just stamped (D7): callers derive deterministic task IDs
	// from it, so a stale zero value would mint colliding IDs.
	return tx.WithContext(ctx).Raw(
		`UPDATE jobs SET title = $1, description = $2, location = $3, employment_type = $4,
		 salary_min = $5, salary_max = $6, currency = $7, required_skills = $8, min_experience = $9,
		 responsibilities = $10, requirements = $11, nice_to_haves = $12, benefits = $13,
		 scoring_weights = $14, min_score_to_proceed = $15, status = $16, proctoring_mode = $17, is_published = $18, rubric = $19, updated_at = NOW()
		 WHERE id = $20
		 RETURNING updated_at`,
		job.Title, job.Description, job.Location, job.EmploymentType,
		job.SalaryMin, job.SalaryMax, job.Currency, reqSkills, job.MinExperience,
		resp, reqs, nice, ben,
		weights, job.MinScoreToProceed, job.Status, job.ProctoringMode, job.IsPublished, job.Rubric, job.ID,
	).Row().Scan(&job.UpdatedAt)
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanJob(row rowScanner) (*jobdomain.Job, error) {
	var (
		j                             jobdomain.Job
		skills, resp, reqs, nice, ben *[]byte
		weights, rubric               []byte
		minScore                      *float64
		minExperience                 *int
		salMin, salMax                *int
		questionSetError              *string
	)
	err := row.Scan(
		&j.ID, &j.OrgID, &j.Title, &j.Description, &j.Location, &j.EmploymentType,
		&salMin, &salMax, &j.Currency,
		&skills, &minExperience, &resp, &reqs, &nice, &ben,
		&weights, &minScore, &j.Status, &j.ProctoringMode, &j.IsPublished, &rubric, &questionSetError, &j.CreatedAt, &j.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, jobdomain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if questionSetError != nil {
		j.QuestionSetError = *questionSetError
	}
	j.SalaryMin = salMin
	j.SalaryMax = salMax
	if minExperience != nil {
		j.MinExperience = *minExperience
	}
	if skills != nil && len(*skills) > 0 && string(*skills) != "null" {
		if err := unmarshalJSONB(&j.RequiredSkills, *skills); err != nil {
			return nil, fmt.Errorf("decode required skills: %w", err)
		}
	}
	if resp != nil && len(*resp) > 0 && string(*resp) != "null" {
		if err := unmarshalJSONB(&j.Responsibilities, *resp); err != nil {
			return nil, fmt.Errorf("decode responsibilities: %w", err)
		}
	}
	if reqs != nil && len(*reqs) > 0 && string(*reqs) != "null" {
		if err := unmarshalJSONB(&j.Requirements, *reqs); err != nil {
			return nil, fmt.Errorf("decode requirements: %w", err)
		}
	}
	if nice != nil && len(*nice) > 0 && string(*nice) != "null" {
		if err := unmarshalJSONB(&j.NiceToHaves, *nice); err != nil {
			return nil, fmt.Errorf("decode nice-to-haves: %w", err)
		}
	}
	if ben != nil && len(*ben) > 0 && string(*ben) != "null" {
		if err := unmarshalJSONB(&j.Benefits, *ben); err != nil {
			return nil, fmt.Errorf("decode benefits: %w", err)
		}
	}
	j.MinScoreToProceed = minScore
	if len(weights) > 0 {
		if err := unmarshalJSONB(&j.ScoringWeights, weights); err != nil {
			return nil, fmt.Errorf("decode scoring weights: %w", err)
		}
	}
	if len(rubric) > 0 && string(rubric) != "null" {
		j.Rubric = rubric
	}
	return &j, nil
}

type PublicJobDTO struct {
	ID               uuid.UUID `json:"id"`
	OrgID            uuid.UUID `json:"org_id"`
	OrgName          string    `json:"org_name"`
	OrgSlug          string    `json:"org_slug"`
	Title            string    `json:"title"`
	Description      string    `json:"description"`
	Location         string    `json:"location"`
	EmploymentType   string    `json:"employment_type"`
	SalaryMin        *int      `json:"salary_min"`
	SalaryMax        *int      `json:"salary_max"`
	Currency         string    `json:"currency"`
	RequiredSkills   []string  `json:"required_skills"`
	MinExperience    int       `json:"min_experience"`
	Responsibilities []string  `json:"responsibilities"`
	Requirements     []string  `json:"requirements"`
	NiceToHaves      []string  `json:"nice_to_haves"`
	Benefits         []string  `json:"benefits"`
	Status           string    `json:"status"`
	ProctoringMode   string    `json:"proctoring_mode"`
	IsPublished      bool      `json:"is_published"`
	Rubric           string    `json:"rubric,omitempty"`
	CreatedAt        string    `json:"created_at"`
}

func (r *PostgresJobRepo) ListPublicActive(ctx context.Context, orgSlug string) ([]*PublicJobDTO, error) {
	rows, err := r.pool.WithContext(ctx).Raw(
		`SELECT `+publicJobColumns+` FROM public_active_jobs_lookup($1)`, orgSlug).Rows()
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	out := []*PublicJobDTO{}
	for rows.Next() {
		j, err := scanPublicJob(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, j)
	}
	return out, rows.Err()
}

func (r *PostgresJobRepo) GetPublicDetail(ctx context.Context, jobID uuid.UUID) (*PublicJobDTO, error) {
	row := r.pool.WithContext(ctx).Raw(
		`SELECT `+publicJobColumns+` FROM public_job_detail_lookup($1)`, jobID).Row()
	j, err := scanPublicJob(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, jobdomain.ErrNotFound
	}
	return j, err
}

// scanPublicJob scans one public job lookup row into a PublicJobDTO. Shared by
// ListPublicActive and GetPublicDetail (identical column order).
func scanPublicJob(row rowScanner) (*PublicJobDTO, error) {
	var (
		j                            PublicJobDTO
		skills, resp, req, nice, ben []byte
		rubric                       []byte
		minExp                       *int
		salMin, salMax               *int
		createdAt                    time.Time
	)
	if err := row.Scan(
		&j.ID, &j.OrgID, &j.OrgName, &j.OrgSlug, &j.Title, &j.Description, &j.Location, &j.EmploymentType,
		&salMin, &salMax, &j.Currency, &skills, &minExp,
		&resp, &req, &nice, &ben, &j.Status, &j.ProctoringMode, &j.IsPublished, &rubric, &createdAt,
	); err != nil {
		return nil, err
	}
	j.CreatedAt = createdAt.Format(time.RFC3339)
	j.SalaryMin = salMin
	j.SalaryMax = salMax
	if minExp != nil {
		j.MinExperience = *minExp
	}
	if err := unmarshalJSONB(&j.RequiredSkills, skills); err != nil {
		return nil, fmt.Errorf("decode public required skills: %w", err)
	}
	if err := unmarshalJSONB(&j.Responsibilities, resp); err != nil {
		return nil, fmt.Errorf("decode public responsibilities: %w", err)
	}
	if err := unmarshalJSONB(&j.Requirements, req); err != nil {
		return nil, fmt.Errorf("decode public requirements: %w", err)
	}
	if err := unmarshalJSONB(&j.NiceToHaves, nice); err != nil {
		return nil, fmt.Errorf("decode public nice-to-haves: %w", err)
	}
	if err := unmarshalJSONB(&j.Benefits, ben); err != nil {
		return nil, fmt.Errorf("decode public benefits: %w", err)
	}
	if len(rubric) > 0 && string(rubric) != "null" {
		j.Rubric = string(rubric)
	}
	return &j, nil
}

// unmarshalJSONB decodes a JSONB column into dst, tolerating the NULL and
// empty representations database/sql surfaces for absent values.
func marshalJSONB[T any](value T) ([]byte, error) {
	return json.Marshal(value)
}

func unmarshalJSONB[T any](dst *T, src []byte) error {
	if len(src) == 0 || string(src) == "null" {
		return nil
	}
	return json.Unmarshal(src, dst)
}

// UpdateRubric — column-scoped rubric write; the full-row Update() would
// clobber concurrent recruiter edits made while the LLM call ran.
func (r *PostgresJobRepo) UpdateRubric(ctx context.Context, id uuid.UUID, rubric json.RawMessage) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	return tx.WithContext(ctx).Exec(
		`UPDATE jobs SET rubric = $1, updated_at = NOW() WHERE id = $2`, string(rubric), id).Error
}

// ListByIDs — batched job fetch for list enrichment (RLS-scoped).
func (r *PostgresJobRepo) ListByIDs(ctx context.Context, orgID uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]*jobdomain.Job, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return map[uuid.UUID]*jobdomain.Job{}, nil
	}
	rows, err := tx.WithContext(ctx).Raw(
		`SELECT id, org_id, title, COALESCE(description, ''), COALESCE(location, ''), employment_type, status, created_at FROM jobs
		 WHERE org_id = $1 AND id = ANY($2)`, orgID, ids).Rows()
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := map[uuid.UUID]*jobdomain.Job{}
	for rows.Next() {
		var j jobdomain.Job
		if err := rows.Scan(&j.ID, &j.OrgID, &j.Title, &j.Description, &j.Location, &j.EmploymentType, &j.Status, &j.CreatedAt); err != nil {
			return nil, err
		}
		out[j.ID] = &j
	}
	return out, rows.Err()
}

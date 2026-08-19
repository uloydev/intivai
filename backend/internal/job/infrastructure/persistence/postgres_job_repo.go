package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	jobdomain "github.com/intivai/backend/internal/job/domain"
	"github.com/intivai/backend/pkg/db"
	"gorm.io/gorm"
)

const jobColumns = `id, org_id, title, description, location, employment_type, salary_min, salary_max, currency, required_skills, min_experience, responsibilities, requirements, nice_to_haves, benefits, scoring_weights, min_score_to_proceed, status, proctoring_mode, is_published, rubric, created_at`

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
	reqSkills, _ := json.Marshal(job.RequiredSkills)
	resp, _ := json.Marshal(job.Responsibilities)
	reqs, _ := json.Marshal(job.Requirements)
	nice, _ := json.Marshal(job.NiceToHaves)
	ben, _ := json.Marshal(job.Benefits)

	return tx.WithContext(ctx).Exec(
		`INSERT INTO jobs (id, org_id, title, description, location, employment_type, salary_min, salary_max, currency,
		                   required_skills, min_experience, responsibilities, requirements, nice_to_haves, benefits,
		                   scoring_weights, min_score_to_proceed, status, proctoring_mode, is_published, rubric, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22)`,
		job.ID, job.OrgID, job.Title, job.Description, job.Location, job.EmploymentType, job.SalaryMin, job.SalaryMax, job.Currency,
		reqSkills, job.MinExperience, resp, reqs, nice, ben,
		job.MarshalScoringWeights(), job.MinScoreToProceed, job.Status, job.ProctoringMode, job.IsPublished, job.Rubric, job.CreatedAt).Error
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
	reqSkills, _ := json.Marshal(job.RequiredSkills)
	resp, _ := json.Marshal(job.Responsibilities)
	reqs, _ := json.Marshal(job.Requirements)
	nice, _ := json.Marshal(job.NiceToHaves)
	ben, _ := json.Marshal(job.Benefits)

	return tx.WithContext(ctx).Exec(
		`UPDATE jobs SET title = $1, description = $2, location = $3, employment_type = $4,
		 salary_min = $5, salary_max = $6, currency = $7, required_skills = $8, min_experience = $9,
		 responsibilities = $10, requirements = $11, nice_to_haves = $12, benefits = $13,
		 scoring_weights = $14, min_score_to_proceed = $15, status = $16, proctoring_mode = $17, is_published = $18, rubric = $19, updated_at = NOW() WHERE id = $20`,
		job.Title, job.Description, job.Location, job.EmploymentType,
		job.SalaryMin, job.SalaryMax, job.Currency, reqSkills, job.MinExperience,
		resp, reqs, nice, ben,
		job.MarshalScoringWeights(), job.MinScoreToProceed, job.Status, job.ProctoringMode, job.IsPublished, job.Rubric, job.ID).Error
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
	)
	err := row.Scan(
		&j.ID, &j.OrgID, &j.Title, &j.Description, &j.Location, &j.EmploymentType,
		&salMin, &salMax, &j.Currency,
		&skills, &minExperience, &resp, &reqs, &nice, &ben,
		&weights, &minScore, &j.Status, &j.ProctoringMode, &j.IsPublished, &rubric, &j.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, jobdomain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	j.SalaryMin = salMin
	j.SalaryMax = salMax
	if minExperience != nil {
		j.MinExperience = *minExperience
	}
	if skills != nil && len(*skills) > 0 && string(*skills) != "null" {
		_ = json.Unmarshal(*skills, &j.RequiredSkills)
	}
	if resp != nil && len(*resp) > 0 && string(*resp) != "null" {
		_ = json.Unmarshal(*resp, &j.Responsibilities)
	}
	if reqs != nil && len(*reqs) > 0 && string(*reqs) != "null" {
		_ = json.Unmarshal(*reqs, &j.Requirements)
	}
	if nice != nil && len(*nice) > 0 && string(*nice) != "null" {
		_ = json.Unmarshal(*nice, &j.NiceToHaves)
	}
	if ben != nil && len(*ben) > 0 && string(*ben) != "null" {
		_ = json.Unmarshal(*ben, &j.Benefits)
	}
	j.MinScoreToProceed = minScore
	if len(weights) > 0 {
		_ = json.Unmarshal(weights, &j.ScoringWeights)
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
	unmarshalJSONB(&j.RequiredSkills, skills)
	unmarshalJSONB(&j.Responsibilities, resp)
	unmarshalJSONB(&j.Requirements, req)
	unmarshalJSONB(&j.NiceToHaves, nice)
	unmarshalJSONB(&j.Benefits, ben)
	if len(rubric) > 0 && string(rubric) != "null" {
		j.Rubric = string(rubric)
	}
	return &j, nil
}

// unmarshalJSONB decodes a JSONB column into dst, tolerating the NULL and
// empty representations database/sql surfaces for absent values.
func unmarshalJSONB(dst any, src []byte) {
	if len(src) > 0 && string(src) != "null" {
		_ = json.Unmarshal(src, dst)
	}
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

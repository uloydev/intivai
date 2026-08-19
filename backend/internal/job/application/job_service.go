package application

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/intivai/backend/internal/iam/application"
	iamdomain "github.com/intivai/backend/internal/iam/domain"
	jobdomain "github.com/intivai/backend/internal/job/domain"
	sharederr "github.com/intivai/backend/internal/shared/errors"
	"github.com/intivai/backend/pkg/queue"
)

type CreateJobCommand struct {
	Title             string
	Description       string
	Location          string
	EmploymentType    string
	SalaryMin         *int
	SalaryMax         *int
	Currency          string
	RequiredSkills    []string
	MinExperience     int
	Responsibilities  []string
	Requirements      []string
	NiceToHaves       []string
	Benefits          []string
	ScoringWeights    map[string]float64
	MinScoreToProceed *float64
}

// UpdateJobCommand — pointer fields: nil = keep current value (PATCH
// semantics; partial updates must not clobber unset fields).
type UpdateJobCommand struct {
	JobID             uuid.UUID
	Title             *string
	Description       *string
	Location          *string
	EmploymentType    *string
	SalaryMin         *int
	SalaryMax         *int
	Currency          *string
	RequiredSkills    *[]string
	MinExperience     *int
	Responsibilities  *[]string
	Requirements      *[]string
	NiceToHaves       *[]string
	Benefits          *[]string
	ScoringWeights    map[string]float64 // nil = keep
	MinScoreToProceed *float64           // nil = keep
	Status            string             // "" = keep
	IsPublished       *bool
}

type JobResult struct {
	ID                uuid.UUID          `json:"id"`
	Title             string             `json:"title"`
	Description       string             `json:"description"`
	Location          string             `json:"location"`
	EmploymentType    string             `json:"employment_type"`
	SalaryMin         *int               `json:"salary_min,omitempty"`
	SalaryMax         *int               `json:"salary_max,omitempty"`
	Currency          string             `json:"currency"`
	RequiredSkills    []string           `json:"required_skills"`
	MinExperience     int                `json:"min_experience"`
	Responsibilities  []string           `json:"responsibilities,omitempty"`
	Requirements      []string           `json:"requirements,omitempty"`
	NiceToHaves       []string           `json:"nice_to_haves,omitempty"`
	Benefits          []string           `json:"benefits,omitempty"`
	ScoringWeights    map[string]float64 `json:"scoring_weights,omitempty"`
	MinScoreToProceed *float64           `json:"min_score_to_proceed,omitempty"`
	Status            string             `json:"status"`
	ProctoringMode    string             `json:"proctoring_mode"`
	IsPublished       bool               `json:"is_published"`
	Rubric            string             `json:"rubric,omitempty"`
	CreatedAt         time.Time          `json:"created_at"`
}

type JobService struct {
	repo jobdomain.JobRepository
	q    *queue.Client
}

func NewJobService(repo jobdomain.JobRepository, queueClient *queue.Client) *JobService {
	return &JobService{repo: repo, q: queueClient}
}

func (s *JobService) Create(ctx context.Context, actor application.AuthContext, cmd CreateJobCommand) (*JobResult, error) {
	if err := application.Authorize(actor, iamdomain.RoleAdmin, iamdomain.RoleRecruiter); err != nil {
		return nil, err
	}
	job, err := jobdomain.NewJob(actor.OrgID, strings.TrimSpace(cmd.Title), strings.TrimSpace(cmd.Description), cmd.RequiredSkills, cmd.MinExperience)
	if err != nil {
		return nil, err
	}
	if cmd.Location != "" {
		job.Location = cmd.Location
	}
	if cmd.EmploymentType != "" {
		job.EmploymentType = cmd.EmploymentType
	}
	job.SalaryMin = cmd.SalaryMin
	job.SalaryMax = cmd.SalaryMax
	if cmd.Currency != "" {
		job.Currency = cmd.Currency
	}
	if len(cmd.Responsibilities) > 0 {
		job.Responsibilities = cmd.Responsibilities
	}
	if len(cmd.Requirements) > 0 {
		job.Requirements = cmd.Requirements
	}
	if len(cmd.NiceToHaves) > 0 {
		job.NiceToHaves = cmd.NiceToHaves
	}
	if len(cmd.Benefits) > 0 {
		job.Benefits = cmd.Benefits
	}
	if err := job.SetScoringWeights(cmd.ScoringWeights); err != nil {
		return nil, err
	}
	job.MinScoreToProceed = cmd.MinScoreToProceed
	if err := s.repo.Create(ctx, job); err != nil {
		return nil, err
	}

	if err := s.enqueueRubric(ctx, job.OrgID, job.ID); err != nil {
		return nil, err
	}

	return toResult(job), nil
}

func (s *JobService) Update(ctx context.Context, actor application.AuthContext, cmd UpdateJobCommand) (*JobResult, error) {
	if err := application.Authorize(actor, iamdomain.RoleAdmin, iamdomain.RoleRecruiter); err != nil {
		return nil, err
	}
	job, err := s.repo.GetByID(ctx, cmd.JobID)
	if errors.Is(err, jobdomain.ErrNotFound) {
		return nil, sharederr.NewNotFoundError("job", cmd.JobID.String())
	}
	if err != nil {
		return nil, err
	}
	if job.OrgID != actor.OrgID {
		return nil, sharederr.NewDomainError("FORBIDDEN", "job belongs to another org")
	}
	if cmd.Title != nil {
		job.Title = strings.TrimSpace(*cmd.Title)
	}
	if cmd.Description != nil {
		job.Description = strings.TrimSpace(*cmd.Description)
	}
	if cmd.RequiredSkills != nil {
		job.RequiredSkills = *cmd.RequiredSkills
	}
	if cmd.MinExperience != nil {
		job.MinExperience = *cmd.MinExperience
	}
	if cmd.Location != nil {
		job.Location = strings.TrimSpace(*cmd.Location)
	}
	if cmd.EmploymentType != nil {
		job.EmploymentType = strings.TrimSpace(*cmd.EmploymentType)
	}
	if cmd.SalaryMin != nil {
		job.SalaryMin = cmd.SalaryMin
	}
	if cmd.SalaryMax != nil {
		job.SalaryMax = cmd.SalaryMax
	}
	if cmd.Currency != nil {
		job.Currency = strings.TrimSpace(*cmd.Currency)
	}
	if cmd.Responsibilities != nil {
		job.Responsibilities = *cmd.Responsibilities
	}
	if cmd.Requirements != nil {
		job.Requirements = *cmd.Requirements
	}
	if cmd.NiceToHaves != nil {
		job.NiceToHaves = *cmd.NiceToHaves
	}
	if cmd.Benefits != nil {
		job.Benefits = *cmd.Benefits
	}
	if cmd.MinScoreToProceed != nil {
		job.MinScoreToProceed = cmd.MinScoreToProceed
	}
	if cmd.Status != "" {
		job.Status = cmd.Status
	}
	if cmd.IsPublished != nil {
		job.IsPublished = *cmd.IsPublished
	}
	if cmd.ScoringWeights != nil {
		if err := job.SetScoringWeights(cmd.ScoringWeights); err != nil {
			return nil, err
		}
	}
	if err := jobdomain.ValidateJobFields(job.Title, job.Description, job.MinExperience, job.Status); err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, job); err != nil {
		return nil, err
	}

	if err := s.enqueueRubric(ctx, job.OrgID, job.ID); err != nil {
		return nil, err
	}

	return toResult(job), nil
}

func (s *JobService) Get(ctx context.Context, actor application.AuthContext, id uuid.UUID) (*JobResult, error) {
	job, err := s.repo.GetByID(ctx, id)
	if errors.Is(err, jobdomain.ErrNotFound) {
		return nil, sharederr.NewNotFoundError("job", id.String())
	}
	if err != nil {
		return nil, err
	}
	if job.OrgID != actor.OrgID {
		return nil, sharederr.NewDomainError("FORBIDDEN", "job belongs to another org")
	}
	return toResult(job), nil
}

func (s *JobService) List(ctx context.Context, actor application.AuthContext) ([]*JobResult, error) {
	jobs, err := s.repo.List(ctx, actor.OrgID)
	if err != nil {
		return nil, err
	}
	out := make([]*JobResult, 0, len(jobs))
	for _, j := range jobs {
		out = append(out, toResult(j))
	}
	return out, nil
}

func toResult(j *jobdomain.Job) *JobResult {
	return &JobResult{
		ID:                j.ID,
		Title:             j.Title,
		Description:       j.Description,
		Location:          j.Location,
		EmploymentType:    j.EmploymentType,
		SalaryMin:         j.SalaryMin,
		SalaryMax:         j.SalaryMax,
		Currency:          j.Currency,
		RequiredSkills:    j.RequiredSkills,
		MinExperience:     j.MinExperience,
		Responsibilities:  j.Responsibilities,
		Requirements:      j.Requirements,
		NiceToHaves:       j.NiceToHaves,
		Benefits:          j.Benefits,
		ScoringWeights:    j.ScoringWeights,
		MinScoreToProceed: j.MinScoreToProceed,
		Status:            j.Status,
		ProctoringMode:    j.ProctoringMode,
		IsPublished:       j.IsPublished,
		Rubric:            string(j.Rubric),
		CreatedAt:         j.CreatedAt,
	}
}

// enqueueRubric — async rubric generation for a job (bounded retries; a
// failed enqueue must not fail the job create/update itself).
func (s *JobService) enqueueRubric(ctx context.Context, orgID, jobID uuid.UUID) error {
	if s.q == nil {
		return nil
	}
	_, err := s.q.Enqueue(ctx, TaskGenerateRubric, GenerateRubricPayload{JobID: jobID.String(), OrgID: orgID.String()}, asynq.MaxRetry(5))
	return err
}

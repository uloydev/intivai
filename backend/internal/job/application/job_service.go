package application

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/intivai/backend/internal/iam/application"
	iamdomain "github.com/intivai/backend/internal/iam/domain"
	jobdomain "github.com/intivai/backend/internal/job/domain"
	"github.com/intivai/backend/internal/shared/errors"
)

type CreateJobCommand struct {
	Title             string
	Description       string
	RequiredSkills    []string
	MinExperience     int
	ScoringWeights    map[string]float64
	MinScoreToProceed *float64
}

// UpdateJobCommand — pointer fields: nil = keep current value (PATCH
// semantics; partial updates must not clobber unset fields).
type UpdateJobCommand struct {
	JobID             uuid.UUID
	Title             *string
	Description       *string
	RequiredSkills    *[]string
	MinExperience     *int
	ScoringWeights    map[string]float64 // nil = keep
	MinScoreToProceed *float64           // nil = keep
	Status            string             // "" = keep
}

type JobResult struct {
	ID                uuid.UUID          `json:"id"`
	Title             string             `json:"title"`
	Description       string             `json:"description"`
	RequiredSkills    []string           `json:"required_skills"`
	MinExperience     int                `json:"min_experience"`
	ScoringWeights    map[string]float64 `json:"scoring_weights,omitempty"`
	MinScoreToProceed *float64           `json:"min_score_to_proceed,omitempty"`
	Status            string             `json:"status"`
	CreatedAt         time.Time          `json:"created_at"`
}

type JobService struct {
	repo jobdomain.JobRepository
}

func NewJobService(repo jobdomain.JobRepository) *JobService {
	return &JobService{repo: repo}
}

func (s *JobService) Create(ctx context.Context, actor application.AuthContext, cmd CreateJobCommand) (*JobResult, error) {
	if err := application.Authorize(actor, iamdomain.RoleAdmin, iamdomain.RoleRecruiter); err != nil {
		return nil, err
	}
	job, err := jobdomain.NewJob(actor.OrgID, strings.TrimSpace(cmd.Title), strings.TrimSpace(cmd.Description), cmd.RequiredSkills, cmd.MinExperience)
	if err != nil {
		return nil, err
	}
	if err := job.SetScoringWeights(cmd.ScoringWeights); err != nil {
		return nil, err
	}
	job.MinScoreToProceed = cmd.MinScoreToProceed
	if err := s.repo.Create(ctx, job); err != nil {
		return nil, err
	}
	return toResult(job), nil
}

func (s *JobService) Update(ctx context.Context, actor application.AuthContext, cmd UpdateJobCommand) (*JobResult, error) {
	if err := application.Authorize(actor, iamdomain.RoleAdmin, iamdomain.RoleRecruiter); err != nil {
		return nil, err
	}
	job, err := s.repo.GetByID(ctx, cmd.JobID)
	if err == jobdomain.ErrNotFound {
		return nil, errors.NewNotFoundError("job", cmd.JobID.String())
	}
	if err != nil {
		return nil, err
	}
	if job.OrgID != actor.OrgID {
		return nil, errors.NewDomainError("FORBIDDEN", "job belongs to another org")
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
	if cmd.MinScoreToProceed != nil {
		job.MinScoreToProceed = cmd.MinScoreToProceed
	}
	if cmd.Status != "" {
		job.Status = cmd.Status
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
	return toResult(job), nil
}

func (s *JobService) Get(ctx context.Context, actor application.AuthContext, id uuid.UUID) (*JobResult, error) {
	job, err := s.repo.GetByID(ctx, id)
	if err == jobdomain.ErrNotFound {
		return nil, errors.NewNotFoundError("job", id.String())
	}
	if err != nil {
		return nil, err
	}
	if job.OrgID != actor.OrgID {
		return nil, errors.NewDomainError("FORBIDDEN", "job belongs to another org")
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
		RequiredSkills:    j.RequiredSkills,
		MinExperience:     j.MinExperience,
		ScoringWeights:    j.ScoringWeights,
		MinScoreToProceed: j.MinScoreToProceed,
		Status:            j.Status,
		CreatedAt:         j.CreatedAt,
	}
}

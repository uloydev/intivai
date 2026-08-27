package application

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/intivai/backend/internal/iam/application"
	iamdomain "github.com/intivai/backend/internal/iam/domain"
	jobdomain "github.com/intivai/backend/internal/job/domain"
	"github.com/intivai/backend/internal/llm"
	sharederr "github.com/intivai/backend/internal/shared/errors"
	"github.com/intivai/backend/pkg/db"
	"gorm.io/gorm"
)

// CandidateContextService — per-job candidate Q&A context (D2). Recruiters
// author it; an AI Suggest drafts from the JD but is NEVER auto-persisted
// (trust rule G10 — recruiter edits + saves separately).
type CandidateContextService struct {
	pool    *gorm.DB
	repo    jobdomain.CandidateContextRepository
	jobRepo jobdomain.JobRepository
	llm     llm.Provider
}

func NewCandidateContextService(pool *gorm.DB, repo jobdomain.CandidateContextRepository, jobRepo jobdomain.JobRepository, llmClient llm.Provider) *CandidateContextService {
	return &CandidateContextService{pool: pool, repo: repo, jobRepo: jobRepo, llm: llmClient}
}

type SaveCandidateContextCommand struct {
	Content string
}

type CandidateContextResult struct {
	JobID     string `json:"job_id"`
	Content   string `json:"content"`
	Version   int    `json:"version"`
	UpdatedAt string `json:"updated_at"`
}

func toCandidateContextResult(c *jobdomain.CandidateContext) *CandidateContextResult {
	return &CandidateContextResult{
		JobID:     c.JobID.String(),
		Content:   c.Content,
		Version:   c.Version,
		UpdatedAt: c.UpdatedAt.Format(time.RFC3339),
	}
}

// Save validates (incl. injection rail) and upserts the context for a job.
// Ownership is enforced inside the tenant tx: the job must belong to the actor.
func (s *CandidateContextService) Save(ctx context.Context, actor application.AuthContext, jobID uuid.UUID, cmd SaveCandidateContextCommand) (*CandidateContextResult, error) {
	if err := application.Authorize(actor, iamdomain.RoleAdmin, iamdomain.RoleRecruiter); err != nil {
		return nil, err
	}
	cc, err := jobdomain.NewCandidateContext(actor.OrgID, jobID, cmd.Content)
	if err != nil {
		return nil, err
	}

	var saved *jobdomain.CandidateContext
	err = db.RunInTx(ctx, s.pool, actor.OrgID.String(), func(tctx context.Context) error {
		job, err := s.jobRepo.GetByID(tctx, jobID)
		if errors.Is(err, jobdomain.ErrNotFound) {
			return sharederr.NewNotFoundError("job", jobID.String())
		}
		if err != nil {
			return err
		}
		if job.OrgID != actor.OrgID {
			return sharederr.NewDomainError("FORBIDDEN", "job belongs to another org")
		}
		if err := s.repo.Upsert(tctx, cc); err != nil {
			return err
		}
		saved = cc
		return nil
	})
	if err != nil {
		return nil, err
	}
	return toCandidateContextResult(saved), nil
}

// Get returns the current context for a job (or 404). Ownership enforced by
// RLS + explicit org check.
func (s *CandidateContextService) Get(ctx context.Context, actor application.AuthContext, jobID uuid.UUID) (*CandidateContextResult, error) {
	if err := application.Authorize(actor, iamdomain.RoleAdmin, iamdomain.RoleRecruiter); err != nil {
		return nil, err
	}
	var cc *jobdomain.CandidateContext
	err := db.RunInTx(ctx, s.pool, actor.OrgID.String(), func(tctx context.Context) error {
		var e error
		cc, e = s.repo.GetByJobID(tctx, jobID)
		return e
	})
	if errors.Is(err, jobdomain.ErrNotFound) {
		return nil, sharederr.NewNotFoundError("candidate context", jobID.String())
	}
	if err != nil {
		return nil, err
	}
	if cc.OrgID != actor.OrgID {
		return nil, sharederr.NewDomainError("FORBIDDEN", "candidate context belongs to another org")
	}
	return toCandidateContextResult(cc), nil
}

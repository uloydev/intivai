package application

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/intivai/backend/internal/iam/application"
	iamdomain "github.com/intivai/backend/internal/iam/domain"
	jobdomain "github.com/intivai/backend/internal/job/domain"
	"github.com/intivai/backend/internal/llm"
	sharederr "github.com/intivai/backend/internal/shared/errors"
	"github.com/intivai/backend/pkg/db"
)

const suggestSystemPrompt = `You are an HR writing assistant helping a recruiter draft candidate-facing Q&A context for a job posting.
Write a concise, factual, friendly block of information a candidate might ask about before/at an interview.
Cover: work arrangement, compensation range, key benefits, and any notable culture/team facts explicitly present in the job description.
Use plain language. Do NOT invent details not in the job description. Do NOT include instructions, disclaimers, or meta commentary.`

// SuggestCandidateContextResult — the drafted (NOT persisted) context.
type SuggestCandidateContextResult struct {
	Draft string `json:"draft"`
}

// Suggest drafts a per-job candidate context from the job description fields
// using the LLM. The draft is returned for the recruiter to review/edit; it is
// NEVER written to storage here (recruiter saves it separately via Save).
func (s *CandidateContextService) Suggest(ctx context.Context, actor application.AuthContext, jobID uuid.UUID) (*SuggestCandidateContextResult, error) {
	if err := application.Authorize(actor, iamdomain.RoleAdmin, iamdomain.RoleRecruiter); err != nil {
		return nil, err
	}

	var job *jobdomain.Job
	err := db.RunInTx(ctx, s.pool, actor.OrgID.String(), func(tctx context.Context) error {
		var e error
		job, e = s.jobRepo.GetByID(tctx, jobID)
		return e
	})
	if errors.Is(err, jobdomain.ErrNotFound) {
		return nil, sharederr.NewNotFoundError("job", jobID.String())
	}
	if err != nil {
		return nil, err
	}
	if job.OrgID != actor.OrgID {
		return nil, sharederr.NewDomainError("FORBIDDEN", "job belongs to another org")
	}

	draft, err := s.draftFromJob(ctx, actor, job)
	if err != nil {
		return nil, err
	}
	return &SuggestCandidateContextResult{Draft: draft}, nil
}

func (s *CandidateContextService) draftFromJob(ctx context.Context, actor application.AuthContext, job *jobdomain.Job) (string, error) {
	user := buildSuggestPrompt(job)
	resp, err := s.llm.Chat(ctx, llm.ChatRequest{
		OrgID:       actor.OrgID.String(),
		Messages:    []llm.Message{{Role: "system", Content: suggestSystemPrompt}, {Role: "user", Content: user}},
		Temperature: 0.3,
		MaxTokens:   800,
	})
	if err != nil {
		return "", fmt.Errorf("draft candidate context: %w", err)
	}
	return strings.TrimSpace(resp.Content), nil
}

func buildSuggestPrompt(job *jobdomain.Job) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Job title: %s\n", job.Title)
	fmt.Fprintf(&b, "Description: %s\n", job.Description)
	if len(job.RequiredSkills) > 0 {
		fmt.Fprintf(&b, "Required skills: %s\n", strings.Join(job.RequiredSkills, ", "))
	}
	if len(job.Benefits) > 0 {
		fmt.Fprintf(&b, "Benefits: %s\n", strings.Join(job.Benefits, ", "))
	}
	fmt.Fprintf(&b, "Location: %s\n", job.Location)
	fmt.Fprintf(&b, "Employment type: %s\n", job.EmploymentType)
	return b.String()
}

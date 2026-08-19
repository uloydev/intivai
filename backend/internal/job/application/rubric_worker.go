package application

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/intivai/backend/internal/job/domain"
	"github.com/intivai/backend/internal/llm"
	"github.com/intivai/backend/pkg/db"
	"github.com/rs/zerolog"
	"gorm.io/gorm"
)

const TaskGenerateRubric = "generate_rubric"

type GenerateRubricPayload struct {
	JobID string `json:"job_id"`
	OrgID string `json:"org_id"`
}

type RubricWorker struct {
	pool      *gorm.DB
	repo      domain.JobRepository
	llmClient *llm.Client
	log       zerolog.Logger
}

func NewRubricWorker(pool *gorm.DB, repo domain.JobRepository, llmClient *llm.Client, log zerolog.Logger) *RubricWorker {
	return &RubricWorker{pool: pool, repo: repo, llmClient: llmClient, log: log}
}

func (w *RubricWorker) Register(mux *asynq.ServeMux) {
	mux.HandleFunc(TaskGenerateRubric, w.handle)
}

func (w *RubricWorker) handle(ctx context.Context, t *asynq.Task) error {
	var p GenerateRubricPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return asynq.SkipRetry
	}
	id, err := uuid.Parse(p.JobID)
	if err != nil {
		return asynq.SkipRetry
	}

	var job *domain.Job
	err = db.RunInTx(ctx, w.pool, p.OrgID, func(txCtx context.Context) error {
		var txErr error
		job, txErr = w.repo.GetByID(txCtx, id)
		if txErr != nil {
			if txErr == domain.ErrNotFound {
				return asynq.SkipRetry
			}
			return txErr
		}
		return nil
	})
	if err != nil {
		return err
	}

	// Skip if rubric already generated
	if len(job.Rubric) > 0 && string(job.Rubric) != "null" {
		return nil
	}

	sys := `You are an expert HR Recruiter and Technical Assessor.
Your task is to generate a Transparent Scoring Rubric based on a Job Description.
This rubric explains to both HR and candidates exactly how their CVs and Interviews will be evaluated.

Output JSON exactly matching the following schema:
{
  "summary": "Brief 1-2 sentence overview of what a successful candidate looks like.",
  "dimensions": [
    {
      "name": "string (e.g., Technical Skills, Experience, Education, Semantic Match)",
      "description": "string (What exactly is being measured here)",
      "weight_percentage": "number (e.g., 35)",
      "criteria": ["string (e.g., Must have 5+ years in Go)"]
    }
  ]
}`

	req, _ := json.Marshal(map[string]interface{}{
		"title":           job.Title,
		"description":     job.Description,
		"required_skills": job.RequiredSkills,
		"min_experience":  job.MinExperience,
		"weights":         job.ScoringWeights, // might be empty/null, that's fine
	})
	user := string(req)

	type RubricSchema struct {
		Summary    string `json:"summary"`
		Dimensions []struct {
			Name             string   `json:"name"`
			Description      string   `json:"description"`
			WeightPercentage int      `json:"weight_percentage"`
			Criteria         []string `json:"criteria"`
		} `json:"dimensions"`
	}

	var schema RubricSchema
	out, err := w.llmClient.StructuredOutput(ctx, llm.StructuredRequest{
		OrgID:  job.OrgID.String(),
		Model:  "multi-qa-MiniLM-L6-cos-v1", // or appropriate model
		System: sys,
		User:   user,
		Schema: schema,
	})
	if err != nil {
		return err
	}

	b, err := json.Marshal(out)
	if err != nil {
		return err
	}

	return db.RunInTx(ctx, w.pool, p.OrgID, func(txCtx context.Context) error {
		// Column-scoped write: the full-row Update() would clobber any
		// recruiter edit made while the LLM call was running.
		return w.repo.UpdateRubric(txCtx, id, b)
	})
}

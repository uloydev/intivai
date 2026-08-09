package application

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	evalllm "github.com/intivai/backend/internal/evaluation/infrastructure/llm"
	ivdomain "github.com/intivai/backend/internal/interview/domain"
	"github.com/intivai/backend/pkg/db"
	"gorm.io/gorm"
)

// TaskEvaluateInterview — async evaluation retry for interviews whose inline
// LLM evaluation failed (candidate did not wait / provider error).
const TaskEvaluateInterview = "evaluate_interview"

type EvaluatePayload struct {
	OrgID       string `json:"org_id"`
	InterviewID string `json:"interview_id"`
}

// EvaluationWorker — asynq handler for TaskEvaluateInterview. Idempotent:
// skips when the interview already has an evaluation (inline path won).
type EvaluationWorker struct {
	pool      *gorm.DB
	ivRepo    ivdomain.InterviewRepository
	evaluator *evalllm.Evaluator
}

func NewEvaluationWorker(pool *gorm.DB, ivRepo ivdomain.InterviewRepository, evaluator *evalllm.Evaluator) *EvaluationWorker {
	return &EvaluationWorker{pool: pool, ivRepo: ivRepo, evaluator: evaluator}
}

func (w *EvaluationWorker) Register(mux *asynq.ServeMux) {
	mux.HandleFunc(TaskEvaluateInterview, w.handle)
}

func (w *EvaluationWorker) handle(ctx context.Context, t *asynq.Task) error {
	var p EvaluatePayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return asynq.SkipRetry
	}
	if p.OrgID == "" || p.InterviewID == "" {
		return asynq.SkipRetry
	}
	ivID, err := uuid.Parse(p.InterviewID)
	if err != nil {
		return asynq.SkipRetry
	}

	var skipped bool
	err = db.RunInTx(ctx, w.pool, p.OrgID, func(tctx context.Context) error {
		iv, err := w.ivRepo.GetByID(tctx, ivID)
		if err != nil {
			return err
		}
		if len(iv.Evaluation) > 0 {
			skipped = true // inline evaluation already won
			return nil
		}
		var pairs []evalllm.TranscriptPair
		for _, a := range iv.Answers {
			if a.Idx < 1 || a.Idx > len(iv.Questions) {
				continue
			}
			q := iv.Questions[a.Idx-1]
			pairs = append(pairs, evalllm.TranscriptPair{
				Idx: a.Idx, Category: q.Category, Question: q.Content, Answer: a.Content,
			})
		}
		if len(pairs) == 0 {
			return ivdomain.ErrNotFound // nothing to evaluate
		}
		report, err := w.evaluator.Evaluate(tctx, pairs)
		if err != nil {
			return err
		}
		raw, err := json.Marshal(report)
		if err != nil {
			return err
		}
		return w.ivRepo.SaveEvaluation(tctx, ivID, raw)
	})
	if skipped {
		return asynq.SkipRetry
	}
	if err == ivdomain.ErrNotFound {
		return asynq.SkipRetry
	}
	return err // transient → asynq retries
}

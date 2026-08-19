package application

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	evalllm "github.com/intivai/backend/internal/evaluation/infrastructure/llm"
	ivdomain "github.com/intivai/backend/internal/interview/domain"
	"github.com/intivai/backend/pkg/db"
	"github.com/intivai/backend/pkg/queue"
	"gorm.io/gorm"
)

// TaskEvaluateInterview — async evaluation retry for interviews whose inline
// LLM evaluation failed (candidate did not wait / provider error).
const TaskEvaluateInterview = queue.TaskEvaluateInterview

type EvaluatePayload struct {
	OrgID       string `json:"org_id"`
	InterviewID string `json:"interview_id"`
}

// EvaluationWorker — asynq handler for TaskEvaluateInterview. Idempotent:
// skips when the interview already has an evaluation (inline path won).
// The LLM call runs OUTSIDE the DB transaction — a held pool connection for
// the full DeepSeek round-trip starves the pool under load.
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

	// Phase 1: read state (short tx — no LLM inside).
	var (
		alreadyEvaluated bool
		pairs            []ivdomain.TranscriptPair
	)
	err = db.RunInTx(ctx, w.pool, p.OrgID, func(tctx context.Context) error {
		iv, err := w.ivRepo.GetByID(tctx, ivID)
		if err != nil {
			return err
		}
		if len(iv.Evaluation) > 0 {
			alreadyEvaluated = true // inline evaluation already won
			return nil
		}
		pairs = iv.TranscriptPairs()
		return nil
	})
	if err != nil {
		if err == ivdomain.ErrNotFound {
			return asynq.SkipRetry
		}
		return err // transient → asynq retries
	}
	if alreadyEvaluated || len(pairs) == 0 {
		return asynq.SkipRetry
	}

	// Phase 2: LLM evaluation (no tx held).
	report, err := w.evaluator.Evaluate(ctx, p.OrgID, pairs)
	if err != nil {
		return err
	}
	raw, err := json.Marshal(report)
	if err != nil {
		return err
	}

	// Phase 3: persist (short tx). ErrEvaluationExists = the inline path won
	// while we were computing — its report stands, ours is discarded.
	err = db.RunInTx(ctx, w.pool, p.OrgID, func(tctx context.Context) error {
		return w.ivRepo.SaveEvaluation(tctx, ivID, raw)
	})
	if err == ivdomain.ErrEvaluationExists {
		return nil
	}
	return err
}

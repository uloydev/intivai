package application

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	evaldomain "github.com/intivai/backend/internal/evaluation/domain"
	evalllm "github.com/intivai/backend/internal/evaluation/infrastructure/llm"
	ivdomain "github.com/intivai/backend/internal/interview/domain"
	notifapp "github.com/intivai/backend/internal/notification/application"
	"github.com/intivai/backend/pkg/db"
	"github.com/intivai/backend/pkg/queue"
	"github.com/rs/zerolog"
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
	queue     *queue.Client
	publicURL string
	log       zerolog.Logger
}

func NewEvaluationWorker(pool *gorm.DB, ivRepo ivdomain.InterviewRepository, evaluator *evalllm.Evaluator, q *queue.Client, publicURL string, log zerolog.Logger) *EvaluationWorker {
	return &EvaluationWorker{pool: pool, ivRepo: ivRepo, evaluator: evaluator, queue: q, publicURL: publicURL, log: log}
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
	if err != nil {
		return err
	}

	// Phase 4: notify the org's recruiters that the scorecard is ready.
	if orgUUID, perr := uuid.Parse(p.OrgID); perr == nil {
		w.notifyScorecard(ctx, orgUUID, ivID, report)
	}
	return nil
}

// notifyScorecard — best-effort scorecard-ready email to org admins/recruiters.
// A failed enqueue must not fail the evaluation itself.
func (w *EvaluationWorker) notifyScorecard(ctx context.Context, orgID, ivID uuid.UUID, report evaldomain.Report) {
	if w.queue == nil {
		return
	}
	recruiters, err := w.recruiterEmails(ctx, orgID)
	if err != nil || len(recruiters) == 0 {
		return
	}
	reportURL := fmt.Sprintf("%s/interviews/%s", strings.TrimSuffix(w.publicURL, "/"), ivID)
	for _, to := range recruiters {
		if _, err := w.queue.Enqueue(ctx, notifapp.TaskSendEmail, notifapp.SendEmailPayload{
			Type:           notifapp.EmailTypeScorecard,
			To:             to,
			CandidateName:  "Candidate", // enriched by the notification payload contract
			JobTitle:       "",
			Score:          report.OverallScore,
			Recommendation: report.Recommendation,
			ReportURL:      reportURL,
		}, asynq.MaxRetry(5)); err != nil {
			w.log.Warn().Err(err).Str("to", to).Msg("enqueue scorecard email failed")
			return
		}
	}
}

// recruiterEmails — the org's admin/recruiter addresses (scorecard audience).
func (w *EvaluationWorker) recruiterEmails(ctx context.Context, orgID uuid.UUID) ([]string, error) {
	rows, err := w.pool.WithContext(ctx).Raw(
		`SELECT email FROM users WHERE org_id = $1 AND role IN ('admin', 'recruiter') AND email <> ''`, orgID).Rows()
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []string
	for rows.Next() {
		var e string
		if err := rows.Scan(&e); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

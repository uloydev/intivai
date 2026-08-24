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
	intdomain "github.com/intivai/backend/internal/integration/domain"
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

// Enqueuer — queue seam (queue.Client satisfies it implicitly); lets tests
// capture enqueued notifications without Redis.
type Enqueuer interface {
	Enqueue(ctx context.Context, jobType string, payload any, opts ...asynq.Option) (*asynq.TaskInfo, error)
}

// EvaluationWorker — asynq handler for TaskEvaluateInterview. Idempotent:
// skips when the interview already has an evaluation (inline path won).
// The LLM call runs OUTSIDE the DB transaction — a held pool connection for
// the full LLM round-trip starves the pool under load.
type EvaluationWorker struct {
	pool      *gorm.DB
	ivRepo    ivdomain.InterviewRepository
	evaluator *evalllm.Evaluator
	queue     Enqueuer
	publicURL string
	webhookFn func(ctx context.Context, orgID uuid.UUID, event intdomain.WebhookEvent, payload []byte) // optional webhook dispatch
	log       zerolog.Logger
}

func NewEvaluationWorker(pool *gorm.DB, ivRepo ivdomain.InterviewRepository, evaluator *evalllm.Evaluator, q Enqueuer, publicURL string, log zerolog.Logger) *EvaluationWorker {
	return &EvaluationWorker{pool: pool, ivRepo: ivRepo, evaluator: evaluator, queue: q, publicURL: publicURL, log: log}
}

func (w *EvaluationWorker) WithWebhookDispatch(fn func(ctx context.Context, orgID uuid.UUID, event intdomain.WebhookEvent, payload []byte)) *EvaluationWorker {
	w.webhookFn = fn
	return w
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

	// Phase 5: dispatch webhooks for interview.completed event.
	if w.webhookFn != nil {
		if orgUUID, perr := uuid.Parse(p.OrgID); perr == nil {
			payload, err := json.Marshal(intdomain.InterviewCompletedPayload{
				InterviewID:    p.InterviewID,
				Score:          report.OverallScore,
				Recommendation: report.Recommendation,
				ReportURL:      fmt.Sprintf("%s/interviews/%s", strings.TrimSuffix(w.publicURL, "/"), ivID),
			})
			if err != nil {
				w.log.Warn().Err(err).Str("interview_id", p.InterviewID).Msg("webhook dispatch: marshal payload failed")
			} else {
				w.webhookFn(ctx, orgUUID, intdomain.EventInterviewCompleted, payload)
			}
		}
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
	if err != nil {
		w.log.Warn().Err(err).Str("org_id", orgID.String()).Msg("resolve scorecard recipients failed")
		return
	}
	if len(recruiters) == 0 {
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
// users is FORCED RLS: the query MUST run inside a tenant tx (app.org_id set),
// otherwise the org predicate matches nothing and recipients silently vanish.
func (w *EvaluationWorker) recruiterEmails(ctx context.Context, orgID uuid.UUID) ([]string, error) {
	var out []string
	err := db.RunInTx(ctx, w.pool, orgID.String(), func(tctx context.Context) error {
		tx, ok := db.TxFrom(tctx)
		if !ok {
			return fmt.Errorf("recruiter emails: no transaction in context")
		}
		rows, err := tx.WithContext(tctx).Raw(
			`SELECT email FROM users WHERE org_id = $1 AND role IN ('admin', 'recruiter') AND email <> ''`, orgID).Rows()
		if err != nil {
			return fmt.Errorf("query scorecard recipients: %w", err)
		}
		defer func() { _ = rows.Close() }()
		for rows.Next() {
			var e string
			if err := rows.Scan(&e); err != nil {
				return fmt.Errorf("scan scorecard recipient: %w", err)
			}
			out = append(out, e)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

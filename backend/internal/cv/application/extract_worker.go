package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/hibiken/asynq"
	cvdomain "github.com/intivai/backend/internal/cv/domain"
	jobdomain "github.com/intivai/backend/internal/job/domain"
	"github.com/intivai/backend/internal/llm"
	memapp "github.com/intivai/backend/internal/memory/application"
	scrapp "github.com/intivai/backend/internal/screening/application"
	scrdomain "github.com/intivai/backend/internal/screening/domain"
	"github.com/intivai/backend/pkg/db"
	"github.com/intivai/backend/pkg/queue"
	"github.com/rs/zerolog"
	"gorm.io/gorm"
)

const (
	TaskExtractCV = "extract_cv"
	maxExtractLen = 12000
)

// ExtractWorker: DeepSeek structured extraction → persist ResumeData, then
// fan out score_cv per active job + sync candidate into the Mnemosyne bank.
type ExtractWorker struct {
	pool     *gorm.DB
	candRepo cvdomain.CandidateRepository
	appRepo  scrdomain.ApplicationRepository
	jobRepo  jobdomain.JobRepository
	llm      llm.Provider
	queue    *queue.Client
	log      zerolog.Logger
}

func NewExtractWorker(pool *gorm.DB, candRepo cvdomain.CandidateRepository, appRepo scrdomain.ApplicationRepository, jobRepo jobdomain.JobRepository, llmClient llm.Provider, q *queue.Client, log zerolog.Logger) *ExtractWorker {
	return &ExtractWorker{pool: pool, candRepo: candRepo, appRepo: appRepo, jobRepo: jobRepo, llm: llmClient, queue: q, log: log}
}

func (w *ExtractWorker) Register(mux *asynq.ServeMux) {
	mux.HandleFunc(TaskExtractCV, w.handle)
}

func (w *ExtractWorker) handle(ctx context.Context, t *asynq.Task) error {
	var p ExtractCVPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return asynq.SkipRetry
	}

	candidate, err := w.loadAndMarkExtracting(ctx, p)
	if err != nil {
		return err
	}

	resume, err := w.extract(ctx, candidate)
	if err != nil {
		return w.fail(ctx, p, err)
	}
	structured, _ := json.Marshal(resume)

	appIDs := []string{}
	err = db.RunInTx(ctx, w.pool, p.OrgID, func(tctx context.Context) error {
		c, err := w.candRepo.GetByID(tctx, candidate.ID)
		if err != nil {
			return err
		}
		c.CVStructured = structured
		c.Status = cvdomain.StatusExtracted
		c.ErrorMessage = ""
		if err := w.candRepo.Update(tctx, c); err != nil {
			return err
		}
		jobs, err := w.jobRepo.ListActive(tctx, candidate.OrgID)
		if err != nil {
			return err
		}
		for _, job := range jobs {
			app := scrdomain.NewApplication(candidate.OrgID, candidate.ID, job.ID)
			if err := w.appRepo.Create(tctx, app); err != nil {
				if scrdomain.IsExists(err) {
					continue
				}
				return err
			}
			appIDs = append(appIDs, app.ID.String())
		}
		return nil
	})
	if err != nil {
		return err
	}

	for _, appID := range appIDs {
		if _, err := w.queue.Enqueue(ctx, scrapp.TaskScoreCV, scrapp.ScoreCVPayload{
			OrgID: p.OrgID, ApplicationID: appID,
		}); err != nil {
			w.log.Error().Err(err).Str("app_id", appID).Msg("enqueue score_cv failed")
		}
	}

	summary := fmt.Sprintf("Candidate %s. Skills: %s. Experience: %.1f years. %s",
		candidate.Name, strings.Join(resume.Skills, ", "), resume.ExperienceYears, resume.Summary)
	if _, err := w.queue.Enqueue(ctx, memapp.TaskSyncMnemosyne, memapp.SyncPayload{
		OrgID:      p.OrgID,
		EntityType: "candidate_profile",
		Summary:    summary,
		Importance: 0.9,
	}); err != nil {
		w.log.Error().Err(err).Msg("enqueue sync_mnemosyne failed")
	}
	return nil
}

func (w *ExtractWorker) loadAndMarkExtracting(ctx context.Context, p ExtractCVPayload) (*cvdomain.Candidate, error) {
	var candidate *cvdomain.Candidate
	err := db.RunInTx(ctx, w.pool, p.OrgID, func(tctx context.Context) error {
		var err error
		candidate, err = w.candRepo.GetByID(tctx, mustUUID(p.CandidateID))
		if errors.Is(err, cvdomain.ErrNotFound) {
			return asynq.SkipRetry
		}
		if err != nil {
			return err
		}
		// Idempotency: a retry after commit must not re-run the LLM or
		// duplicate bank syncs.
		if candidate.Status == cvdomain.StatusExtracted {
			return asynq.SkipRetry
		}
		candidate.Status = cvdomain.StatusExtracting
		return w.candRepo.Update(tctx, candidate)
	})
	return candidate, err
}

func (w *ExtractWorker) extract(ctx context.Context, candidate *cvdomain.Candidate) (*ResumeData, error) {
	user := candidate.CVRawText
	if len(user) > maxExtractLen {
		user = user[:maxExtractLen]
	}
	schema := &ResumeData{}
	out, err := w.llm.StructuredOutput(ctx, llm.StructuredRequest{
		System: "Extract structured resume data from the CV text. Return ONLY valid JSON matching the schema. Use empty arrays and zero values when data is missing.",
		User:   user,
		Schema: schema,
	})
	if err != nil {
		return nil, fmt.Errorf("extract llm: %w", err)
	}
	rd, ok := out.(*ResumeData)
	if !ok || rd == nil {
		return nil, errors.New("extract llm returned unexpected shape")
	}
	return rd, nil
}

func (w *ExtractWorker) fail(ctx context.Context, p ExtractCVPayload, cause error) error {
	uer := db.RunInTx(ctx, w.pool, p.OrgID, func(tctx context.Context) error {
		c, err := w.candRepo.GetByID(tctx, mustUUID(p.CandidateID))
		if err != nil {
			return err
		}
		c.Status = cvdomain.StatusFailedExtract
		c.ErrorMessage = cause.Error()
		return w.candRepo.Update(tctx, c)
	})
	if uer != nil {
		w.log.Error().Err(uer).Msg("extract fail update")
	}
	w.log.Error().Err(cause).Str("candidate_id", p.CandidateID).Msg("extract_cv failed")
	return asynq.SkipRetry
}

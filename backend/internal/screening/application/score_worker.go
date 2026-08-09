package application

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	cvdomain "github.com/intivai/backend/internal/cv/domain"
	jobdomain "github.com/intivai/backend/internal/job/domain"
	scrdomain "github.com/intivai/backend/internal/screening/domain"
	"github.com/intivai/backend/pkg/db"
	"github.com/rs/zerolog"
	"gorm.io/gorm"
)

const TaskScoreCV = "score_cv"

type ScoreCVPayload struct {
	OrgID         string `json:"org_id"`
	ApplicationID string `json:"application_id"`
}

// ScoreWorker runs the weighted screening algorithm for one application.
type ScoreWorker struct {
	pool     *gorm.DB
	appRepo  scrdomain.ApplicationRepository
	candRepo cvdomain.CandidateRepository
	jobRepo  jobdomain.JobRepository
	orgs     OrgSettingsReader
	log      zerolog.Logger
}

func NewScoreWorker(pool *gorm.DB, appRepo scrdomain.ApplicationRepository, candRepo cvdomain.CandidateRepository, jobRepo jobdomain.JobRepository, orgs OrgSettingsReader, log zerolog.Logger) *ScoreWorker {
	return &ScoreWorker{pool: pool, appRepo: appRepo, candRepo: candRepo, jobRepo: jobRepo, orgs: orgs, log: log}
}

func (w *ScoreWorker) Register(mux *asynq.ServeMux) {
	mux.HandleFunc(TaskScoreCV, w.handle)
}

func (w *ScoreWorker) handle(ctx context.Context, t *asynq.Task) error {
	var p ScoreCVPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return asynq.SkipRetry
	}
	appID := mustUUID(p.ApplicationID)

	return db.RunInTx(ctx, w.pool, p.OrgID, func(tctx context.Context) error {
		app, err := w.appRepo.GetByID(tctx, appID)
		if scrdomain.IsNotFound(err) {
			return asynq.SkipRetry
		}
		if err != nil {
			return err
		}

		candidate, err := w.candRepo.GetByID(tctx, app.CandidateID)
		if err != nil {
			return err
		}
		job, err := w.jobRepo.GetByID(tctx, app.JobID)
		if err != nil {
			return err
		}
		orgWeights, orgMin, err := w.orgs.ReadOrgSettings(tctx, app.OrgID)
		if err != nil {
			return err
		}

		resume, err := resumeFromCandidate(candidate)
		if err != nil {
			return asynq.SkipRetry
		}

		semantic := scrdomain.SemanticScore(
			candidate.CVRawText+" "+resume.Summary+" "+strings.Join(resume.Skills, " "),
			job.Description+" "+strings.Join(job.RequiredSkills, " "),
		)

		result := scrdomain.Score(resume, scrdomain.JobInfo{
			RequiredSkills:    job.RequiredSkills,
			MinExperience:     job.MinExperience,
			MinScoreToProceed: jobMin(job),
			ScoringWeights:    job.ScoringWeights,
		}, scrdomain.OrgInfo{ScoringWeights: orgWeights, MinScoreToProceed: orgMin}, semantic)

		breakdown, _ := json.Marshal(result.Breakdown)
		app.CVScore = &result.Total
		app.ScoreBreakdown = breakdown
		app.PassedScreening = &result.Passed
		if result.Passed {
			app.Status = scrdomain.StatusPassed
		} else {
			app.Status = scrdomain.StatusRejected
		}
		return w.appRepo.Update(tctx, app)
	})
}

func resumeFromCandidate(c *cvdomain.Candidate) (scrdomain.ResumeData, error) {
	var r scrdomain.ResumeData
	if len(c.CVStructured) == 0 {
		return r, errors.New("candidate has no structured data")
	}
	if err := json.Unmarshal(c.CVStructured, &r); err != nil {
		return r, err
	}
	return r, nil
}

func jobMin(j *jobdomain.Job) float64 {
	if j.MinScoreToProceed != nil {
		return *j.MinScoreToProceed
	}
	return 0
}

func mustUUID(s string) uuid.UUID {
	id, _ := uuid.Parse(s)
	return id
}

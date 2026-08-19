package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	cvdomain "github.com/intivai/backend/internal/cv/domain"
	"github.com/intivai/backend/internal/iam/application"
	iamdomain "github.com/intivai/backend/internal/iam/domain"
	jobdomain "github.com/intivai/backend/internal/job/domain"
	notifapp "github.com/intivai/backend/internal/notification/application"
	scrdomain "github.com/intivai/backend/internal/screening/domain"
	sharederr "github.com/intivai/backend/internal/shared/errors"
	"github.com/intivai/backend/pkg/db"
	"github.com/intivai/backend/pkg/queue"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

// ScreeningService creates applications (candidate × job) and triggers
// async scoring. Used by POST /screenings and the extract pipeline.
type ScreeningService struct {
	pool      *gorm.DB
	appRepo   scrdomain.ApplicationRepository
	candRepo  cvdomain.CandidateRepository
	jobRepo   jobdomain.JobRepository
	queue     *queue.Client
	portalURL string
}

func NewScreeningService(pool *gorm.DB, appRepo scrdomain.ApplicationRepository, candRepo cvdomain.CandidateRepository, jobRepo jobdomain.JobRepository, queueClient *queue.Client, portalURL string) *ScreeningService {
	return &ScreeningService{pool: pool, appRepo: appRepo, candRepo: candRepo, jobRepo: jobRepo, queue: queueClient, portalURL: strings.TrimSuffix(portalURL, "/")}
}

type CreateScreeningCommand struct {
	CandidateID uuid.UUID
	JobID       uuid.UUID
}

type ApplicationResult struct {
	ID              uuid.UUID       `json:"id"`
	CandidateID     uuid.UUID       `json:"candidate_id"`
	CandidateName   string          `json:"candidate_name"`
	CandidateEmail  string          `json:"candidate_email"`
	JobID           uuid.UUID       `json:"job_id"`
	JobTitle        string          `json:"job_title"`
	Status          string          `json:"status"`
	CVScore         *float64        `json:"cv_score,omitempty"`
	PassedScreening *bool           `json:"passed_screening,omitempty"`
	Stage           *string         `json:"stage,omitempty"`
	RecruiterNotes  *string         `json:"recruiter_notes,omitempty"`
	ScoreBreakdown  json.RawMessage `json:"score_breakdown,omitempty"`
	InterviewScore  *float64        `json:"interview_score,omitempty"`
}

func (s *ScreeningService) Create(ctx context.Context, actor application.AuthContext, cmd CreateScreeningCommand) (*ApplicationResult, error) {
	if err := application.Authorize(actor, iamdomain.RoleAdmin, iamdomain.RoleRecruiter); err != nil {
		return nil, err
	}
	var app *scrdomain.Application
	err := db.RunInTx(ctx, s.pool, actor.OrgID.String(), func(tctx context.Context) error {
		candidate, err := s.candRepo.GetByID(tctx, cmd.CandidateID)
		if errors.Is(err, cvdomain.ErrNotFound) {
			return sharederr.NewNotFoundError("candidate", cmd.CandidateID.String())
		}
		if err != nil {
			return err
		}
		if candidate.OrgID != actor.OrgID {
			return sharederr.NewDomainError("FORBIDDEN", "candidate belongs to another org")
		}
		if len(candidate.CVStructured) == 0 {
			return sharederr.NewDomainError("CANDIDATE_NOT_READY", "candidate has no structured data yet")
		}
		job, err := s.jobRepo.GetByID(tctx, cmd.JobID)
		if errors.Is(err, jobdomain.ErrNotFound) {
			return sharederr.NewNotFoundError("job", cmd.JobID.String())
		}
		if err != nil {
			return err
		}
		if job.OrgID != actor.OrgID {
			return sharederr.NewDomainError("FORBIDDEN", "job belongs to another org")
		}
		if job.Status != jobdomain.StatusActive {
			return sharederr.NewDomainError("JOB_NOT_ACTIVE", "job is not active")
		}

		app, err = s.appRepo.GetByCandidateJob(tctx, actor.OrgID, cmd.CandidateID, cmd.JobID)
		if err == nil {
			return nil // already exists — re-trigger score below
		}
		if err != scrdomain.ErrNotFound {
			return err
		}

		tx, ok := db.TxFrom(tctx)
		if !ok {
			return db.ErrNoTx
		}
		newApp := scrdomain.NewApplication(actor.OrgID, cmd.CandidateID, cmd.JobID)
		winning, _, err := CreateApplicationWithRecovery(tctx, tx, s.appRepo, newApp)
		if err != nil {
			return err
		}
		app = winning
		return nil
	})
	if err != nil {
		return nil, err
	}
	if _, err := s.queue.Enqueue(ctx, TaskScoreCV, ScoreCVPayload{
		OrgID: actor.OrgID.String(), ApplicationID: app.ID.String(),
	}, asynq.MaxRetry(5)); err != nil {
		return nil, err
	}
	return &ApplicationResult{ID: app.ID, CandidateID: app.CandidateID, JobID: app.JobID, Status: app.Status}, nil
}

// UpdateDecision persists the recruiter lifecycle stage + hiring notes
// (PATCH semantics: nil field = keep current value). Stage transitions follow
// the ladder (ADR-0001): forward moves + terminal states from anywhere; a
// backward/correction move requires an admin.
func (s *ScreeningService) UpdateDecision(ctx context.Context, actor application.AuthContext, appID uuid.UUID, stage, notes *string) (*ApplicationResult, error) {
	if err := application.Authorize(actor, iamdomain.RoleAdmin, iamdomain.RoleRecruiter); err != nil {
		return nil, err
	}
	var next *scrdomain.Stage
	if stage != nil {
		trimmed := strings.TrimSpace(*stage)
		st := scrdomain.Stage(trimmed)
		if !st.IsValid() {
			return nil, sharederr.NewDomainError("INVALID_STAGE", "unknown lifecycle stage")
		}
		next = &st
	}
	var out *ApplicationResult
	err := db.RunInTx(ctx, s.pool, actor.OrgID.String(), func(tctx context.Context) error {
		app, err := s.appRepo.GetByID(tctx, appID)
		if err != nil {
			if errors.Is(err, scrdomain.ErrNotFound) {
				return sharederr.NewNotFoundError("application", appID.String())
			}
			return err
		}
		if next != nil {
			fromNil := app.Stage == nil
			var current scrdomain.Stage
			if app.Stage != nil {
				current = scrdomain.Stage(*app.Stage)
			}
			if !current.CanTransitionTo(*next, fromNil) {
				// Backward/correction moves are allowed only for admins (ADR-0001).
				if !current.RequiresAdmin(*next) || actor.Role != string(iamdomain.RoleAdmin) {
					return sharederr.NewDomainError("INVALID_STAGE_TRANSITION",
						fmt.Sprintf("transition from %q to %q is not allowed", current, *next))
				}
			}
		}
		if err := s.appRepo.UpdateDecision(tctx, actor.OrgID, appID, next, notes); err != nil {
			return err
		}
		app, err = s.appRepo.GetByID(tctx, appID)
		if err != nil {
			return err
		}
		out = &ApplicationResult{
			ID: app.ID, CandidateID: app.CandidateID, JobID: app.JobID, Status: app.Status,
			CVScore:         app.CVScore,
			PassedScreening: app.PassedScreening,
			Stage:           app.Stage,
			RecruiterNotes:  app.RecruiterNotes,
			ScoreBreakdown:  app.ScoreBreakdown,
		}
		return nil
	})
	if err != nil {
		return out, err
	}

	// Candidate-facing decision emails (ADR-0001 terminal stages). A failed
	// enqueue must not roll back the persisted decision — log and continue.
	if next != nil && (*next == scrdomain.StageOfferExtended || *next == scrdomain.StageRejected) {
		s.notifyDecision(ctx, actor.OrgID, appID, *next)
	}

	return out, nil
}

// notifyDecision — best-effort candidate decision email (offer extended /
// rejected). Looks up candidate + job details for the message.
func (s *ScreeningService) notifyDecision(ctx context.Context, orgID uuid.UUID, appID uuid.UUID, stage scrdomain.Stage) {
	if s.queue == nil {
		return
	}
	email, name, jobTitle, err := s.decisionDetails(ctx, orgID, appID)
	if err != nil || email == "" {
		return
	}
	label := "Offer extended"
	if stage == scrdomain.StageRejected {
		label = "Application not proceeding"
	}
	if _, err := s.queue.Enqueue(ctx, notifapp.TaskSendEmail, notifapp.SendEmailPayload{
		Type:          notifapp.EmailTypeCandidateDecision,
		To:            email,
		CandidateName: name,
		JobTitle:      jobTitle,
		Decision:      label,
		PortalURL:     s.portalURL + "/candidate/portal",
	}, asynq.MaxRetry(5)); err != nil {
		s.logError("enqueue decision email failed", err)
	}
}

func (s *ScreeningService) decisionDetails(ctx context.Context, orgID uuid.UUID, appID uuid.UUID) (email, name, jobTitle string, err error) {
	err = db.RunInTx(ctx, s.pool, orgID.String(), func(tctx context.Context) error {
		app, err := s.appRepo.GetByID(tctx, appID)
		if err != nil {
			return err
		}
		cand, err := s.candRepo.GetByID(tctx, app.CandidateID)
		if err != nil {
			return err
		}
		job, err := s.jobRepo.GetByID(tctx, app.JobID)
		if err != nil {
			return err
		}
		email, name, jobTitle = cand.Email, cand.Name, job.Title
		return nil
	})
	return email, name, jobTitle, err
}

func (s *ScreeningService) logError(msg string, err error) {
	log.Warn().Err(err).Msg(msg)
}

func (s *ScreeningService) List(ctx context.Context, actor application.AuthContext, jobID uuid.UUID) ([]*ApplicationResult, error) {
	var out []*ApplicationResult
	err := db.RunInTx(ctx, s.pool, actor.OrgID.String(), func(tctx context.Context) error {
		apps, err := s.appRepo.List(tctx, actor.OrgID, jobID)
		if err != nil {
			return err
		}
		out = make([]*ApplicationResult, 0, len(apps))
		// Batch lookups (2 queries + maps) instead of 2×N GetByID round-trips.
		cands, err := s.candRepo.ListByIDs(tctx, actor.OrgID, appCandidateIDs(apps))
		if err != nil {
			return err
		}
		jobs, err := s.jobRepo.ListByIDs(tctx, actor.OrgID, appJobIDs(apps))
		if err != nil {
			return err
		}
		for _, a := range apps {
			r := &ApplicationResult{
				ID: a.ID, CandidateID: a.CandidateID, JobID: a.JobID, Status: a.Status,
				CVScore: a.CVScore, PassedScreening: a.PassedScreening,
				Stage: a.Stage, RecruiterNotes: a.RecruiterNotes,
				ScoreBreakdown: a.ScoreBreakdown,
				InterviewScore: a.InterviewScore,
			}
			if c, ok := cands[a.CandidateID]; ok {
				r.CandidateName = c.Name
				r.CandidateEmail = c.Email
			}
			if j, ok := jobs[a.JobID]; ok {
				r.JobTitle = j.Title
			}
			out = append(out, r)
		}
		return nil
	})
	return out, err
}

func appCandidateIDs(apps []*scrdomain.Application) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(apps))
	for _, a := range apps {
		ids = append(ids, a.CandidateID)
	}
	return ids
}

func appJobIDs(apps []*scrdomain.Application) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(apps))
	for _, a := range apps {
		ids = append(ids, a.JobID)
	}
	return ids
}

package application

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	cvdomain "github.com/intivai/backend/internal/cv/domain"
	"github.com/intivai/backend/internal/iam/application"
	iamdomain "github.com/intivai/backend/internal/iam/domain"
	jobdomain "github.com/intivai/backend/internal/job/domain"
	scrdomain "github.com/intivai/backend/internal/screening/domain"
	"github.com/intivai/backend/internal/shared/errors"
	"github.com/intivai/backend/pkg/db"
	"github.com/intivai/backend/pkg/queue"
	"gorm.io/gorm"
)

// ScreeningService creates applications (candidate × job) and triggers
// async scoring. Used by POST /screenings and the extract pipeline.
type ScreeningService struct {
	pool     *gorm.DB
	appRepo  scrdomain.ApplicationRepository
	candRepo cvdomain.CandidateRepository
	jobRepo  jobdomain.JobRepository
	queue    *queue.Client
}

func NewScreeningService(pool *gorm.DB, appRepo scrdomain.ApplicationRepository, candRepo cvdomain.CandidateRepository, jobRepo jobdomain.JobRepository, q *queue.Client) *ScreeningService {
	return &ScreeningService{pool: pool, appRepo: appRepo, candRepo: candRepo, jobRepo: jobRepo, queue: q}
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
		if err == cvdomain.ErrNotFound {
			return errors.NewNotFoundError("candidate", cmd.CandidateID.String())
		}
		if err != nil {
			return err
		}
		if candidate.OrgID != actor.OrgID {
			return errors.NewDomainError("FORBIDDEN", "candidate belongs to another org")
		}
		if len(candidate.CVStructured) == 0 {
			return errors.NewDomainError("CANDIDATE_NOT_READY", "candidate has no structured data yet")
		}
		job, err := s.jobRepo.GetByID(tctx, cmd.JobID)
		if err == jobdomain.ErrNotFound {
			return errors.NewNotFoundError("job", cmd.JobID.String())
		}
		if err != nil {
			return err
		}
		if job.OrgID != actor.OrgID {
			return errors.NewDomainError("FORBIDDEN", "job belongs to another org")
		}
		if job.Status != jobdomain.StatusActive {
			return errors.NewDomainError("JOB_NOT_ACTIVE", "job is not active")
		}

		app, err = s.appRepo.GetByCandidateJob(tctx, actor.OrgID, cmd.CandidateID, cmd.JobID)
		if err == nil {
			return nil // already exists — re-trigger score below
		}
		if err != scrdomain.ErrNotFound {
			return err
		}

		tx, _ := db.TxFrom(tctx)
		app = scrdomain.NewApplication(actor.OrgID, cmd.CandidateID, cmd.JobID)
		if err := tx.SavePoint("create_app").Error; err != nil {
			return err
		}
		if err := s.appRepo.Create(tctx, app); err != nil {
			if scrdomain.IsExists(err) {
				// Concurrent create — 23505 aborts the tx; roll back to the
				// savepoint and reload the winning row.
				if rerr := tx.RollbackTo("create_app").Error; rerr != nil {
					return rerr
				}
				app, err = s.appRepo.GetByCandidateJob(tctx, actor.OrgID, cmd.CandidateID, cmd.JobID)
				return err
			}
			return err
		}
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
			return nil, errors.NewDomainError("INVALID_STAGE", "unknown lifecycle stage")
		}
		next = &st
	}
	var out *ApplicationResult
	err := db.RunInTx(ctx, s.pool, actor.OrgID.String(), func(tctx context.Context) error {
		app, err := s.appRepo.GetByID(tctx, appID)
		if err != nil {
			if err == scrdomain.ErrNotFound {
				return errors.NewNotFoundError("application", appID.String())
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
				return errors.NewDomainError("INVALID_STAGE_TRANSITION",
					fmt.Sprintf("transition from %q to %q is not allowed", current, *next))
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
	return out, err
}

func (s *ScreeningService) List(ctx context.Context, actor application.AuthContext, jobID uuid.UUID) ([]*ApplicationResult, error) {
	var out []*ApplicationResult
	err := db.RunInTx(ctx, s.pool, actor.OrgID.String(), func(tctx context.Context) error {
		apps, err := s.appRepo.List(tctx, actor.OrgID, jobID)
		if err != nil {
			return err
		}
		out = make([]*ApplicationResult, 0, len(apps))
		for _, a := range apps {
			r := &ApplicationResult{
				ID: a.ID, CandidateID: a.CandidateID, JobID: a.JobID, Status: a.Status,
				CVScore: a.CVScore, PassedScreening: a.PassedScreening,
				Stage: a.Stage, RecruiterNotes: a.RecruiterNotes,
				ScoreBreakdown: a.ScoreBreakdown,
			}
			// Candidate/job lookups are RLS-scoped to the tenant tx. NotFound
			// → empty display field; real errors surface loudly instead of
			// silently blanking FE rows.
			if c, err := s.candRepo.GetByID(tctx, a.CandidateID); err == nil {
				r.CandidateName = c.Name
				r.CandidateEmail = c.Email
			} else if err != cvdomain.ErrNotFound {
				return err
			}
			if j, err := s.jobRepo.GetByID(tctx, a.JobID); err == nil {
				r.JobTitle = j.Title
			} else if err != jobdomain.ErrNotFound {
				return err
			}
			out = append(out, r)
		}
		return nil
	})
	return out, err
}

package application

import (
	"context"

	"github.com/google/uuid"
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
	ID              uuid.UUID `json:"id"`
	CandidateID     uuid.UUID `json:"candidate_id"`
	JobID           uuid.UUID `json:"job_id"`
	Status          string    `json:"status"`
	CVScore         *float64  `json:"cv_score,omitempty"`
	PassedScreening *bool     `json:"passed_screening,omitempty"`
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
	}); err != nil {
		return nil, err
	}
	return &ApplicationResult{ID: app.ID, CandidateID: app.CandidateID, JobID: app.JobID, Status: app.Status}, nil
}

func (s *ScreeningService) List(ctx context.Context, actor application.AuthContext, jobID uuid.UUID) ([]*ApplicationResult, error) {
	apps, err := s.appRepo.List(ctx, actor.OrgID, jobID)
	if err != nil {
		return nil, err
	}
	out := make([]*ApplicationResult, 0, len(apps))
	for _, a := range apps {
		out = append(out, &ApplicationResult{
			ID: a.ID, CandidateID: a.CandidateID, JobID: a.JobID, Status: a.Status,
			CVScore: a.CVScore, PassedScreening: a.PassedScreening,
		})
	}
	return out, nil
}

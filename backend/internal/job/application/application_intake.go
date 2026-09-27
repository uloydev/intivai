package application

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	cvapp "github.com/intivai/backend/internal/cv/application"
	cvdomain "github.com/intivai/backend/internal/cv/domain"
	jobdomain "github.com/intivai/backend/internal/job/domain"
	notifapp "github.com/intivai/backend/internal/notification/application"
	scrdomain "github.com/intivai/backend/internal/screening/domain"
	sharederr "github.com/intivai/backend/internal/shared/errors"
	"github.com/intivai/backend/pkg/db"
	"github.com/rs/zerolog"
	"gorm.io/gorm"
)

// portalTokenTTL — a portal magic token minted at apply time stays valid for
// 24h so the candidate can reach the tracker without an OTP round-trip.
const portalTokenTTL = 24 * time.Hour

// MaxApplyPerEmailPerDay — abuse cap: applications per (org, lower(email))
// within 24h. Generous for real candidates, tight for credential-stuffing /
// mass-apply bots hitting the unauthenticated endpoint.
const MaxApplyPerEmailPerDay = 10

// ApplyCapExceeded — pure decision seam for the per-email daily cap.
func ApplyCapExceeded(recent int) bool {
	return recent >= MaxApplyPerEmailPerDay
}

type ApplyCommand struct {
	JobID       uuid.UUID
	Name        string
	Email       string
	Resume      io.Reader
	ResumeSize  int64
	ContentType string
}

type ApplyResult struct {
	CandidateID uuid.UUID `json:"candidate_id"`
	JobID       uuid.UUID `json:"job_id"`
	Status      string    `json:"status"`
	Message     string    `json:"message"`
}

type PublicJobInfo struct {
	ID    uuid.UUID
	OrgID uuid.UUID
	Title string
}

type JobReader interface {
	GetPublicDetail(ctx context.Context, id uuid.UUID) (*PublicJobInfo, error)
}

type ObjectStore interface {
	Upload(ctx context.Context, key string, r io.Reader, size int64, contentType string) error
	Delete(ctx context.Context, key string) error
}

type Enqueuer interface {
	Enqueue(ctx context.Context, jobType string, payload any, opts ...asynq.Option) (*asynq.TaskInfo, error)
}

type MagicTokenCreator interface {
	CreateMagicToken(ctx context.Context, email, token string, expiresAt time.Time) error
}

type ApplicationIntakeService interface {
	Apply(ctx context.Context, cmd ApplyCommand) (*ApplyResult, error)
}

type ApplicationIntakeServiceImpl struct {
	pool       *gorm.DB
	jobRepo    JobReader
	candRepo   cvdomain.CandidateRepository
	appRepo    scrdomain.ApplicationRepository
	store      ObjectStore
	queue      Enqueuer
	portalRepo MagicTokenCreator
	publicURL  string
	log        zerolog.Logger
}

func NewApplicationIntakeService(
	pool *gorm.DB,
	jobRepo JobReader,
	candRepo cvdomain.CandidateRepository,
	appRepo scrdomain.ApplicationRepository,
	store ObjectStore,
	queue Enqueuer,
	portalRepo MagicTokenCreator,
	publicURL string,
	log zerolog.Logger,
) *ApplicationIntakeServiceImpl {
	return &ApplicationIntakeServiceImpl{
		pool:       pool,
		jobRepo:    jobRepo,
		candRepo:   candRepo,
		appRepo:    appRepo,
		store:      store,
		queue:      queue,
		portalRepo: portalRepo,
		publicURL:  publicURL,
		log:        log,
	}
}

func (s *ApplicationIntakeServiceImpl) Apply(ctx context.Context, cmd ApplyCommand) (*ApplyResult, error) {
	if cmd.JobID == uuid.Nil {
		return nil, sharederr.NewDomainError("INVALID_INPUT", "invalid job id")
	}

	name := strings.TrimSpace(cmd.Name)
	email := strings.TrimSpace(cmd.Email)
	if name == "" || email == "" {
		return nil, sharederr.NewDomainError("INVALID_INPUT", "name and email are required")
	}

	if cmd.Resume == nil {
		return nil, sharederr.NewDomainError("INVALID_INPUT", "resume PDF file is required")
	}

	const maxUploadBytes = 10 * 1024 * 1024
	if cmd.ResumeSize > maxUploadBytes {
		return nil, sharederr.NewDomainError("INVALID_INPUT", "file size exceeds 10MB limit")
	}

	fileBytes, err := io.ReadAll(io.LimitReader(cmd.Resume, maxUploadBytes+1))
	if err != nil {
		return nil, sharederr.NewInternal(err)
	}
	if len(fileBytes) > maxUploadBytes {
		return nil, sharederr.NewDomainError("INVALID_INPUT", "file size exceeds 10MB limit")
	}

	if len(fileBytes) < 5 || !bytes.HasPrefix(fileBytes, []byte("%PDF-")) {
		return nil, sharederr.NewDomainError("INVALID_INPUT", "resume must be a valid PDF file")
	}

	job, err := s.jobRepo.GetPublicDetail(ctx, cmd.JobID)
	if errors.Is(err, jobdomain.ErrNotFound) {
		return nil, sharederr.NewNotFoundError("job", cmd.JobID.String())
	}
	if err != nil {
		return nil, sharederr.NewInternal(err)
	}

	if _, err := cvdomain.NewCandidate(job.OrgID, name, email); err != nil {
		return nil, sharederr.NewDomainError("BAD_REQUEST", "valid name and email are required")
	}

	contentType := cmd.ContentType
	if contentType == "" {
		contentType = "application/pdf"
	}

	var candidateID uuid.UUID
	isNewCandidate := false

	runInTx := func(fn func(txCtx context.Context) error) error {
		if s.pool != nil {
			return db.RunInTx(ctx, s.pool, job.OrgID.String(), fn)
		}
		return fn(ctx)
	}

	err = runInTx(func(txCtx context.Context) error {
		recent, cerr := s.appRepo.CountRecentByCandidateEmail(txCtx, job.OrgID, email, time.Now().UTC().Add(-24*time.Hour))
		if cerr != nil {
			return cerr
		}
		if ApplyCapExceeded(recent) {
			return sharederr.NewDomainError("TOO_MANY_REQUESTS", "too many applications from this email today")
		}
		id, isNew, aerr := s.appRepo.ApplyWithDedupe(txCtx, job.OrgID, cmd.JobID, name, email)
		if aerr != nil {
			return aerr
		}
		candidateID = id
		isNewCandidate = isNew
		return nil
	})
	if err != nil {
		var de *sharederr.DomainError
		if errors.As(err, &de) {
			return nil, de
		}
		return nil, sharederr.NewInternal(err)
	}

	candidatePath := fmt.Sprintf("cvs/%s/%s.pdf", job.OrgID, candidateID)
	if err := s.store.Upload(ctx, candidatePath, bytes.NewReader(fileBytes), int64(len(fileBytes)), contentType); err != nil {
		if isNewCandidate {
			cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
			defer cancel()
			if rerr := s.rollbackApply(cleanupCtx, job.OrgID, candidateID, cmd.JobID); rerr != nil {
				s.log.Error().Err(rerr).Str("org_id", job.OrgID.String()).Str("candidate_id", candidateID.String()).Msg("rollback application failed after upload error")
			}
		}
		return nil, sharederr.NewInternal(err)
	}

	if _, err := s.queue.Enqueue(ctx, cvapp.TaskParseCV, cvapp.ParseCVPayload{
		OrgID: job.OrgID.String(), CandidateID: candidateID.String(),
	}, asynq.MaxRetry(5)); err != nil {
		// D4: only delete the object for a NEW candidate. A re-applying
		// candidate's candidatePath IS the existing CV object — deleting it
		// destroys the stored resume. Re-apply keeps CV + row; the failure
		// only surfaces without mutating prior data.
		if isNewCandidate {
			cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
			defer cancel()
			if derr := s.store.Delete(cleanupCtx, candidatePath); derr != nil {
				s.log.Warn().Err(derr).Str("path", candidatePath).Str("org_id", job.OrgID.String()).Str("candidate_id", candidateID.String()).Msg("failed to delete orphan resume during rollback")
			}
			if rerr := s.rollbackApply(cleanupCtx, job.OrgID, candidateID, cmd.JobID); rerr != nil {
				s.log.Error().Err(rerr).Str("org_id", job.OrgID.String()).Str("candidate_id", candidateID.String()).Msg("rollback application failed after enqueue error")
			}
		}
		return nil, sharederr.NewInternal(err)
	}

	if _, err := s.queue.Enqueue(ctx, notifapp.TaskSendEmail, notifapp.SendEmailPayload{
		Type:          notifapp.EmailTypeConfirmation,
		To:            email,
		CandidateName: name,
		JobTitle:      job.Title,
	}, asynq.MaxRetry(5)); err != nil {
		s.log.Warn().Err(err).Str("candidate_id", candidateID.String()).Msg("enqueue confirmation email failed")
	}

	if portalToken := uuid.NewString(); s.portalRepo != nil {
		if err := s.portalRepo.CreateMagicToken(ctx, strings.ToLower(email), portalToken, time.Now().UTC().Add(portalTokenTTL)); err != nil {
			s.log.Error().Err(err).Str("candidate_id", candidateID.String()).Str("email", email).Msg("mint portal magic token failed")
		} else if _, err := s.queue.Enqueue(ctx, notifapp.TaskSendEmail, notifapp.SendEmailPayload{
			Type:          notifapp.EmailTypePortalAccess,
			To:            email,
			CandidateName: name,
			MagicLink:     fmt.Sprintf("%s/candidate/portal?token=%s", strings.TrimSuffix(s.publicURL, "/"), portalToken),
		}, asynq.MaxRetry(5)); err != nil {
			s.log.Warn().Err(err).Str("candidate_id", candidateID.String()).Msg("enqueue portal access email failed")
		}
	}

	return &ApplyResult{
		CandidateID: candidateID,
		JobID:       cmd.JobID,
		Status:      "submitted",
		Message:     "Application received successfully and queued for AI screening",
	}, nil
}

func (s *ApplicationIntakeServiceImpl) rollbackApply(ctx context.Context, orgID, candidateID, jobID uuid.UUID) error {
	run := func(txCtx context.Context) error {
		if s.pool != nil {
			if tx, ok := db.TxFrom(txCtx); ok {
				if err := tx.Exec("DELETE FROM applications WHERE candidate_id = ? AND job_id = ?", candidateID, jobID).Error; err != nil {
					return fmt.Errorf("rollback applications row: %w", err)
				}
			}
		}
		if s.candRepo != nil {
			if err := s.candRepo.Delete(txCtx, candidateID); err != nil {
				return fmt.Errorf("rollback candidate row: %w", err)
			}
		}
		return nil
	}

	if s.pool != nil {
		return db.RunInTx(ctx, s.pool, orgID.String(), run)
	}
	return run(ctx)
}

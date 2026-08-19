package api

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	cvapp "github.com/intivai/backend/internal/cv/application"
	cvdomain "github.com/intivai/backend/internal/cv/domain"
	jobdomain "github.com/intivai/backend/internal/job/domain"
	"github.com/intivai/backend/internal/job/infrastructure/persistence"
	notifapp "github.com/intivai/backend/internal/notification/application"
	scrdomain "github.com/intivai/backend/internal/screening/domain"
	sharederr "github.com/intivai/backend/internal/shared/errors"
	"github.com/intivai/backend/internal/shared/httpapi"
	"github.com/intivai/backend/pkg/db"
	"github.com/intivai/backend/pkg/queue"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

type PublicJobHandler struct {
	pool       *gorm.DB
	jobRepo    *persistence.PostgresJobRepo
	candRepo   cvdomain.CandidateRepository
	appRepo    scrdomain.ApplicationRepository
	store      cvapp.ObjectStore
	queue      *queue.Client
	portalRepo scrdomain.CandidatePortalRepository
}

func NewPublicJobHandler(
	pool *gorm.DB,
	jobRepo *persistence.PostgresJobRepo,
	candRepo cvdomain.CandidateRepository,
	appRepo scrdomain.ApplicationRepository,
	store cvapp.ObjectStore,
	q *queue.Client,
	portalRepo scrdomain.CandidatePortalRepository,
) *PublicJobHandler {
	return &PublicJobHandler{
		pool:       pool,
		jobRepo:    jobRepo,
		candRepo:   candRepo,
		appRepo:    appRepo,
		store:      store,
		queue:      q,
		portalRepo: portalRepo,
	}
}

// ListPublicJobs handles GET /api/v1/public/jobs
func (h *PublicJobHandler) ListPublicJobs(c *fiber.Ctx) error {
	orgSlug := c.Query("org", "")
	jobs, err := h.jobRepo.ListPublicActive(c.UserContext(), orgSlug)
	if err != nil {
		return httpapi.Error(c, sharederr.NewDomainError("INTERNAL_ERROR", "internal server error"))
	}
	return httpapi.OK(c, jobs)
}

// GetPublicJob handles GET /api/v1/public/jobs/:id
func (h *PublicJobHandler) GetPublicJob(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httpapi.Error(c, sharederr.NewDomainError("BAD_REQUEST", "invalid job id"))
	}
	job, err := h.jobRepo.GetPublicDetail(c.UserContext(), id)
	if errors.Is(err, jobdomain.ErrNotFound) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "job not found or inactive"})
	}
	if err != nil {
		return httpapi.Error(c, sharederr.NewDomainError("INTERNAL_ERROR", "internal server error"))
	}
	return httpapi.OK(c, job)
}

// portalTokenTTL — a portal magic token minted at apply time stays valid for
// 24h so the candidate can reach the tracker without an email round-trip.
const portalTokenTTL = 24 * time.Hour

type PublicApplyResponse struct {
	CandidateID uuid.UUID `json:"candidate_id"`
	JobID       uuid.UUID `json:"job_id"`
	Status      string    `json:"status"`
	Message     string    `json:"message"`
	// PortalToken — one-time magic token for /candidate/portal?token=...,
	// exchanged via POST /public/candidate/auth/verify. Empty if minting failed.
	PortalToken string `json:"portal_token"`
}

// Apply handles POST /api/v1/public/jobs/:id/apply (multipart form)
func (h *PublicJobHandler) Apply(c *fiber.Ctx) error {
	jobID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httpapi.Error(c, sharederr.NewDomainError("INVALID_INPUT", "invalid job id"))
	}

	name := strings.TrimSpace(c.FormValue("name"))
	email := strings.TrimSpace(c.FormValue("email"))
	if name == "" || email == "" {
		return httpapi.Error(c, sharederr.NewDomainError("INVALID_INPUT", "name and email are required"))
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		return httpapi.Error(c, sharederr.NewDomainError("INVALID_INPUT", "resume PDF file is required"))
	}

	// 10MB limit
	if fileHeader.Size > 10*1024*1024 {
		return httpapi.Error(c, sharederr.NewDomainError("INVALID_INPUT", "file size exceeds 10MB limit"))
	}

	file, err := fileHeader.Open()
	if err != nil {
		return httpapi.Error(c, sharederr.NewDomainError("INVALID_INPUT", "cannot open uploaded file"))
	}
	defer func() { _ = file.Close() }()

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		return httpapi.Error(c, sharederr.NewDomainError("INTERNAL_ERROR", "cannot read file bytes"))
	}
	// Same validation as the authed upload path: non-empty PDF magic.
	if len(fileBytes) < 5 || !bytes.HasPrefix(fileBytes, []byte("%PDF-")) {
		return httpapi.Error(c, sharederr.NewDomainError("INVALID_INPUT", "resume must be a valid PDF file"))
	}

	// Verify job exists and is active
	job, err := h.jobRepo.GetPublicDetail(c.UserContext(), jobID)
	if errors.Is(err, jobdomain.ErrNotFound) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "job not found or inactive"})
	}
	if err != nil {
		return httpapi.Error(c, sharederr.NewDomainError("INTERNAL_ERROR", "internal server error"))
	}

	// Validate name/email via the CV domain constructor (validation only —
	// the row itself is created by the repo's ApplyWithDedupe).
	if _, err := cvdomain.NewCandidate(job.OrgID, name, email); err != nil {
		return httpapi.Error(c, sharederr.NewDomainError("BAD_REQUEST", "valid name and email are required"))
	}
	contentType := "application/pdf"
	var candidateID uuid.UUID
	isNewCandidate := false

	err = db.RunInTx(c.UserContext(), h.pool, job.OrgID.String(), func(txCtx context.Context) error {
		id, isNew, aerr := h.appRepo.ApplyWithDedupe(txCtx, job.OrgID, jobID, name, email)
		if aerr != nil {
			return aerr
		}
		candidateID = id
		isNewCandidate = isNew
		return nil
	})
	if err != nil {
		return httpapi.Error(c, sharederr.NewDomainError("INTERNAL_ERROR", "failed to save candidate application record"))
	}

	// Store the (possibly updated) resume under the candidate's key — after
	// the DB work so a slow upload never holds the transaction.
	candidatePath := fmt.Sprintf("cvs/%s/%s.pdf", job.OrgID, candidateID)
	if err := h.store.Upload(c.UserContext(), candidatePath, bytes.NewReader(fileBytes), int64(len(fileBytes)), contentType); err != nil {
		if isNewCandidate {
			_ = h.rollbackApply(c, job.OrgID, candidateID, jobID)
		}
		return httpapi.Error(c, sharederr.NewDomainError("INTERNAL_ERROR", "failed to store resume"))
	}

	// Enqueue parse task
	if _, err := h.queue.Enqueue(c.UserContext(), cvapp.TaskParseCV, cvapp.ParseCVPayload{
		OrgID: job.OrgID.String(), CandidateID: candidateID.String(),
	}, asynq.MaxRetry(5)); err != nil {
		_ = h.store.Delete(c.UserContext(), candidatePath)
		if isNewCandidate {
			_ = h.rollbackApply(c, job.OrgID, candidateID, jobID)
		}
		return httpapi.Error(c, sharederr.NewDomainError("INTERNAL_ERROR", "failed to enqueue cv processing"))
	}

	// Enqueue confirmation email
	if _, err := h.queue.Enqueue(c.UserContext(), notifapp.TaskSendEmail, notifapp.SendEmailPayload{
		Type:          notifapp.EmailTypeConfirmation,
		To:            email,
		CandidateName: name,
		JobTitle:      job.Title,
	}, asynq.MaxRetry(5)); err != nil {
		log.Warn().Err(err).Str("candidate_id", candidateID.String()).Msg("enqueue confirmation email failed")
	}

	// Mint a portal magic token so the success screen can land the candidate
	// directly on their own tracker (no OTP email round-trip). Best-effort: a
	// failure must not fail the already-committed application — the response
	// simply carries an empty portal_token and the flow degrades to OTP login.
	portalToken := uuid.NewString()
	if err := h.portalRepo.CreateMagicToken(c.UserContext(), strings.ToLower(email), portalToken, time.Now().UTC().Add(portalTokenTTL)); err != nil {
		log.Error().Err(err).Str("candidate_id", candidateID.String()).Str("email", email).Msg("mint portal magic token failed")
		portalToken = ""
	}

	return httpapi.Created(c, PublicApplyResponse{
		CandidateID: candidateID,
		JobID:       jobID,
		Status:      "submitted",
		Message:     "Application received successfully and queued for AI screening",
		PortalToken: portalToken,
	})
}

// rollbackApply — best-effort removal of a just-created candidate + its
// application when a later step (resume upload / task enqueue) fails.
// Only called for NEW candidates; a re-applying candidate's data is kept.
func (h *PublicJobHandler) rollbackApply(c *fiber.Ctx, orgID, candidateID, jobID uuid.UUID) error {
	return db.RunInTx(c.UserContext(), h.pool, orgID.String(), func(txCtx context.Context) error {
		if tx, ok := db.TxFrom(txCtx); ok {
			_ = tx.Exec("DELETE FROM applications WHERE candidate_id = ? AND job_id = ?", candidateID, jobID)
		}
		return h.candRepo.Delete(txCtx, candidateID)
	})
}

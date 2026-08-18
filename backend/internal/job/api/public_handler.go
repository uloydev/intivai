package api

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	cvapp "github.com/intivai/backend/internal/cv/application"
	cvdomain "github.com/intivai/backend/internal/cv/domain"
	jobdomain "github.com/intivai/backend/internal/job/domain"
	"github.com/intivai/backend/internal/job/infrastructure/persistence"
	notifapp "github.com/intivai/backend/internal/notification/application"
	"github.com/intivai/backend/internal/shared/httpapi"
	"github.com/intivai/backend/pkg/db"
	"github.com/intivai/backend/pkg/queue"
	"gorm.io/gorm"
)

type PublicJobHandler struct {
	pool     *gorm.DB
	jobRepo  *persistence.PostgresJobRepo
	candRepo cvdomain.CandidateRepository
	store    cvapp.ObjectStore
	queue    *queue.Client
}

func NewPublicJobHandler(
	pool *gorm.DB,
	jobRepo *persistence.PostgresJobRepo,
	candRepo cvdomain.CandidateRepository,
	store cvapp.ObjectStore,
	q *queue.Client,
) *PublicJobHandler {
	return &PublicJobHandler{
		pool:     pool,
		jobRepo:  jobRepo,
		candRepo: candRepo,
		store:    store,
		queue:    q,
	}
}

// ListPublicJobs handles GET /api/v1/public/jobs
func (h *PublicJobHandler) ListPublicJobs(c *fiber.Ctx) error {
	orgSlug := c.Query("org", "")
	jobs, err := h.jobRepo.ListPublicActive(c.UserContext(), orgSlug)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal server error"})
	}
	return httpapi.OK(c, jobs)
}

// GetPublicJob handles GET /api/v1/public/jobs/:id
func (h *PublicJobHandler) GetPublicJob(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid job id"})
	}
	job, err := h.jobRepo.GetPublicDetail(c.UserContext(), id)
	if err == jobdomain.ErrNotFound {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "job not found or inactive"})
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal server error"})
	}
	return httpapi.OK(c, job)
}

type PublicApplyResponse struct {
	CandidateID uuid.UUID `json:"candidate_id"`
	JobID       uuid.UUID `json:"job_id"`
	Status      string    `json:"status"`
	Message     string    `json:"message"`
}

// Apply handles POST /api/v1/public/jobs/:id/apply (multipart form)
func (h *PublicJobHandler) Apply(c *fiber.Ctx) error {
	jobID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid job id"})
	}

	name := strings.TrimSpace(c.FormValue("name"))
	email := strings.TrimSpace(c.FormValue("email"))
	if name == "" || email == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "name and email are required"})
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "resume PDF file is required"})
	}

	// 10MB limit
	if fileHeader.Size > 10*1024*1024 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "file size exceeds 10MB limit"})
	}

	file, err := fileHeader.Open()
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "cannot open uploaded file"})
	}
	defer func() { _ = file.Close() }()

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "cannot read file bytes"})
	}
	// Same validation as the authed upload path: non-empty PDF magic.
	if len(fileBytes) < 5 || !bytes.HasPrefix(fileBytes, []byte("%PDF-")) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "resume must be a valid PDF file"})
	}

	// Verify job exists and is active
	job, err := h.jobRepo.GetPublicDetail(c.UserContext(), jobID)
	if err == jobdomain.ErrNotFound {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "job not found or inactive"})
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal server error"})
	}

	candidate := &cvdomain.Candidate{}
	contentType := "application/pdf"

	err = db.RunInTx(c.UserContext(), h.pool, job.OrgID.String(), func(txCtx context.Context) error {
		gormTx, ok := db.TxFrom(txCtx)
		if !ok {
			return db.ErrNoTx
		}
		// Serialize per (org, email): no unique constraint exists on
		// candidates(org_id, email), so re-applies must reuse the SAME
		// candidate instead of orphaned duplicates.
		lockKey := job.OrgID.String() + ":" + strings.ToLower(email)
		if err := gormTx.Exec(`SELECT pg_advisory_xact_lock(hashtext(?))`, lockKey).Error; err != nil {
			return err
		}
		var existingID uuid.UUID
		row := gormTx.Raw(
			`SELECT id FROM candidates WHERE org_id = ? AND LOWER(email) = ? ORDER BY created_at LIMIT 1`,
			job.OrgID, strings.ToLower(email)).Row()
		err := row.Scan(&existingID)
		switch {
		case err == nil:
			candidate, err = h.candRepo.GetByID(txCtx, existingID)
			if err != nil {
				return err
			}
		case errors.Is(err, sql.ErrNoRows):
			candidate, err = cvdomain.NewCandidate(job.OrgID, name, email)
			if err != nil {
				return err
			}
			candidate.CVPath = fmt.Sprintf("cvs/%s/%s.pdf", job.OrgID, candidate.ID)
			candidate.Status = cvdomain.StatusParsing
			if err := h.candRepo.Create(txCtx, candidate); err != nil {
				return err
			}
		default:
			// Real DB failure — must NOT fall through to "create a duplicate".
			return err
		}

		// Store the (possibly updated) resume under the candidate's key
		if err := h.store.Upload(txCtx, candidate.CVPath, bytes.NewReader(fileBytes), int64(len(fileBytes)), contentType); err != nil {
			return err
		}

		var appID uuid.UUID
		return gormTx.Raw(
			`INSERT INTO applications (id, org_id, candidate_id, job_id, status, created_at, updated_at)
			 VALUES (?, ?, ?, ?, 'screening', NOW(), NOW())
			 ON CONFLICT (candidate_id, job_id) DO UPDATE SET updated_at = NOW()
			 RETURNING id`,
			uuid.New(), job.OrgID, candidate.ID, jobID,
		).Row().Scan(&appID)
	})
	if err != nil {
		_ = h.store.Delete(c.UserContext(), candidate.CVPath)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to save candidate application record"})
	}

	// Enqueue parse task
	if _, err := h.queue.Enqueue(c.UserContext(), cvapp.TaskParseCV, cvapp.ParseCVPayload{
		OrgID: job.OrgID.String(), CandidateID: candidate.ID.String(),
	}); err != nil {
		_ = h.store.Delete(c.UserContext(), candidate.CVPath)
		_ = db.RunInTx(c.UserContext(), h.pool, job.OrgID.String(), func(txCtx context.Context) error {
			_ = h.candRepo.Delete(txCtx, candidate.ID)
			return nil
		})
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to enqueue cv processing"})
	}

	// Enqueue confirmation email
	_, _ = h.queue.Enqueue(c.UserContext(), notifapp.TaskSendEmail, notifapp.SendEmailPayload{
		Type:          notifapp.EmailTypeConfirmation,
		To:            candidate.Email,
		CandidateName: candidate.Name,
		JobTitle:      job.Title,
	})

	return httpapi.Created(c, PublicApplyResponse{
		CandidateID: candidate.ID,
		JobID:       jobID,
		Status:      "submitted",
		Message:     "Application received successfully and queued for AI screening",
	})
}

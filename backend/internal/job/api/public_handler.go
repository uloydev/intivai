package api

import (
	"context"
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	cvapp "github.com/intivai/backend/internal/cv/application"
	cvdomain "github.com/intivai/backend/internal/cv/domain"
	jobapp "github.com/intivai/backend/internal/job/application"
	jobdomain "github.com/intivai/backend/internal/job/domain"
	"github.com/intivai/backend/internal/job/infrastructure/persistence"
	scrdomain "github.com/intivai/backend/internal/screening/domain"
	sharederr "github.com/intivai/backend/internal/shared/errors"
	"github.com/intivai/backend/internal/shared/httpapi"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

type PublicJobHandler struct {
	jobRepo       *persistence.PostgresJobRepo
	intakeService jobapp.ApplicationIntakeService
}

// Enqueuer — queue seam (queue.Client satisfies it implicitly); lets tests
// capture enqueued tasks without Redis.
type Enqueuer interface {
	Enqueue(ctx context.Context, jobType string, payload any, opts ...asynq.Option) (*asynq.TaskInfo, error)
}

type jobRepoAdapter struct {
	repo *persistence.PostgresJobRepo
}

func (a *jobRepoAdapter) GetPublicDetail(ctx context.Context, id uuid.UUID) (*jobapp.PublicJobInfo, error) {
	dto, err := a.repo.GetPublicDetail(ctx, id)
	if err != nil {
		return nil, err
	}
	return &jobapp.PublicJobInfo{
		ID:    dto.ID,
		OrgID: dto.OrgID,
		Title: dto.Title,
	}, nil
}

func NewPublicJobHandler(
	pool *gorm.DB,
	jobRepo *persistence.PostgresJobRepo,
	candRepo cvdomain.CandidateRepository,
	appRepo scrdomain.ApplicationRepository,
	store cvapp.ObjectStore,
	q Enqueuer,
	portalRepo scrdomain.CandidatePortalRepository,
	publicURL string,
) *PublicJobHandler {
	intakeService := jobapp.NewApplicationIntakeService(
		pool, &jobRepoAdapter{repo: jobRepo}, candRepo, appRepo, store, q, portalRepo, publicURL, log.Logger,
	)
	return &PublicJobHandler{
		jobRepo:       jobRepo,
		intakeService: intakeService,
	}
}

func NewPublicJobHandlerWithIntake(
	jobRepo *persistence.PostgresJobRepo,
	intakeService jobapp.ApplicationIntakeService,
) *PublicJobHandler {
	return &PublicJobHandler{
		jobRepo:       jobRepo,
		intakeService: intakeService,
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

// MaxApplyPerEmailPerDay — abuse cap: applications per (org, lower(email))
// within 24h. Generous for real candidates, tight for credential-stuffing /
// mass-apply bots hitting the unauthenticated endpoint.
const MaxApplyPerEmailPerDay = jobapp.MaxApplyPerEmailPerDay

// ApplyCapExceeded — pure decision seam for the per-email daily cap.
func ApplyCapExceeded(recent int) bool {
	return jobapp.ApplyCapExceeded(recent)
}

type PublicApplyResponse = jobapp.ApplyResult

// Apply handles POST /api/v1/public/jobs/:id/apply (multipart form)
func (h *PublicJobHandler) Apply(c *fiber.Ctx) error {
	jobID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httpapi.Error(c, sharederr.NewDomainError("INVALID_INPUT", "invalid job id"))
	}

	name := strings.TrimSpace(c.FormValue("name"))
	email := strings.TrimSpace(c.FormValue("email"))

	fileHeader, err := c.FormFile("file")
	if err != nil {
		return httpapi.Error(c, sharederr.NewDomainError("INVALID_INPUT", "resume PDF file is required"))
	}

	file, err := fileHeader.Open()
	if err != nil {
		return httpapi.Error(c, sharederr.NewDomainError("INVALID_INPUT", "cannot open uploaded file"))
	}
	defer func() { _ = file.Close() }()

	res, err := h.intakeService.Apply(c.UserContext(), jobapp.ApplyCommand{
		JobID:       jobID,
		Name:        name,
		Email:       email,
		Resume:      file,
		ResumeSize:  fileHeader.Size,
		ContentType: "application/pdf",
	})
	if err != nil {
		return httpapi.Error(c, err)
	}

	return httpapi.Created(c, res)
}

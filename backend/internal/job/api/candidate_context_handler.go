package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/intivai/backend/internal/iam/api"
	jobapp "github.com/intivai/backend/internal/job/application"
	sharederr "github.com/intivai/backend/internal/shared/errors"
	"github.com/intivai/backend/internal/shared/httpapi"
)

type CandidateContextHandler struct {
	svc *jobapp.CandidateContextService
}

func NewCandidateContextHandler(svc *jobapp.CandidateContextService) *CandidateContextHandler {
	return &CandidateContextHandler{svc: svc}
}

// RegisterRoutes mounts the candidate-context endpoints on the authed group
// (auth + tenant-tx middlewares are applied by the caller). Role enforcement
// (admin/recruiter) lives in the service's Authorize calls.
func (h *CandidateContextHandler) RegisterRoutes(g fiber.Router) {
	g.Get("/jobs/:id/candidate-context", h.Get)
	g.Put("/jobs/:id/candidate-context", h.Save)
	g.Post("/jobs/:id/candidate-context/suggest", h.Suggest)
}

type candidateContextRequest struct {
	Content string `json:"content"`
}

// Save — PUT /api/v1/jobs/:id/candidate-context
func (h *CandidateContextHandler) Save(c *fiber.Ctx) error {
	actor, err := api.RequireActor(c)
	if err != nil {
		return err
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httpapi.Error(c, sharederr.NewDomainError("BAD_REQUEST", "invalid job id"))
	}
	var req candidateContextRequest
	if err := c.BodyParser(&req); err != nil {
		return httpapi.Error(c, sharederr.NewDomainError("BAD_REQUEST", "invalid body"))
	}
	result, err := h.svc.Save(c.UserContext(), actor, id, jobapp.SaveCandidateContextCommand{Content: req.Content})
	if err != nil {
		return httpapi.Error(c, err)
	}
	return httpapi.OK(c, result)
}

// Get — GET /api/v1/jobs/:id/candidate-context
func (h *CandidateContextHandler) Get(c *fiber.Ctx) error {
	actor, err := api.RequireActor(c)
	if err != nil {
		return err
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httpapi.Error(c, sharederr.NewDomainError("BAD_REQUEST", "invalid job id"))
	}
	result, err := h.svc.Get(c.UserContext(), actor, id)
	if err != nil {
		return httpapi.Error(c, err)
	}
	return httpapi.OK(c, result)
}

// Suggest — POST /api/v1/jobs/:id/candidate-context/suggest
// Returns an AI DRAFT only (never persisted). The recruiter edits then saves.
func (h *CandidateContextHandler) Suggest(c *fiber.Ctx) error {
	actor, err := api.RequireActor(c)
	if err != nil {
		return err
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httpapi.Error(c, sharederr.NewDomainError("BAD_REQUEST", "invalid job id"))
	}
	result, err := h.svc.Suggest(c.UserContext(), actor, id)
	if err != nil {
		return httpapi.Error(c, err)
	}
	return httpapi.OK(c, result)
}

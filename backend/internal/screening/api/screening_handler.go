package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/intivai/backend/internal/iam/api"
	scrapp "github.com/intivai/backend/internal/screening/application"
	sharederr "github.com/intivai/backend/internal/shared/errors"
	"github.com/intivai/backend/internal/shared/httpapi"
)

type ScreeningHandler struct {
	svc *scrapp.ScreeningService
}

func NewScreeningHandler(svc *scrapp.ScreeningService) *ScreeningHandler {
	return &ScreeningHandler{svc: svc}
}

type screeningRequest struct {
	CandidateID string `json:"candidate_id"`
	JobID       string `json:"job_id"`
}

func (h *ScreeningHandler) Create(c *fiber.Ctx) error {
	actor, err := api.RequireActor(c)
	if err != nil {
		return err
	}
	var req screeningRequest
	if err := c.BodyParser(&req); err != nil {
		return httpapi.Error(c, sharederr.NewDomainError("BAD_REQUEST", "invalid body"))
	}
	candidateID, err := uuid.Parse(req.CandidateID)
	if err != nil {
		return httpapi.Error(c, sharederr.NewDomainError("BAD_REQUEST", "invalid candidate_id"))
	}
	jobID, err := uuid.Parse(req.JobID)
	if err != nil {
		return httpapi.Error(c, sharederr.NewDomainError("BAD_REQUEST", "invalid job_id"))
	}
	result, err := h.svc.Create(c.UserContext(), actor, scrapp.CreateScreeningCommand{CandidateID: candidateID, JobID: jobID})
	if err != nil {
		return httpapi.Error(c, err)
	}
	return httpapi.Created(c, result)
}

func (h *ScreeningHandler) List(c *fiber.Ctx) error {
	actor, err := api.RequireActor(c)
	if err != nil {
		return err
	}
	var jobID uuid.UUID
	if raw := c.Query("job_id"); raw != "" {
		jobID, err = uuid.Parse(raw)
		if err != nil {
			return httpapi.Error(c, sharederr.NewDomainError("BAD_REQUEST", "invalid job_id"))
		}
	}
	result, err := h.svc.List(c.UserContext(), actor, jobID)
	if err != nil {
		return httpapi.Error(c, err)
	}
	return httpapi.OK(c, result)
}

// UpdateDecision — PATCH /applications/:id (recruiter lifecycle stage +
// notes). PATCH semantics: pointer fields; nil = keep current.
func (h *ScreeningHandler) UpdateDecision(c *fiber.Ctx) error {
	actor, err := api.RequireActor(c)
	if err != nil {
		return err
	}
	appID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httpapi.Error(c, sharederr.NewDomainError("BAD_REQUEST", "invalid application id"))
	}
	var req struct {
		Stage *string `json:"stage"`
		Notes *string `json:"recruiter_notes"`
	}
	if err := c.BodyParser(&req); err != nil {
		return httpapi.Error(c, sharederr.NewDomainError("BAD_REQUEST", "invalid body"))
	}
	if req.Stage == nil && req.Notes == nil {
		return httpapi.Error(c, sharederr.NewDomainError("BAD_REQUEST", "nothing to update"))
	}
	result, err := h.svc.UpdateDecision(c.UserContext(), actor, appID, req.Stage, req.Notes)
	if err != nil {
		return httpapi.Error(c, err)
	}
	return httpapi.OK(c, result)
}

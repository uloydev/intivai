package api

import (
	"github.com/gofiber/fiber/v2"
	iamapi "github.com/intivai/backend/internal/iam/api"
	iamapp "github.com/intivai/backend/internal/iam/application"
	iamdomain "github.com/intivai/backend/internal/iam/domain"
	sbapp "github.com/intivai/backend/internal/sandbox/application"
	"github.com/intivai/backend/internal/sandbox/domain"
	"github.com/intivai/backend/internal/shared/httpapi"
)

type SandboxHandler struct {
	svc *sbapp.SandboxService
}

func NewSandboxHandler(svc *sbapp.SandboxService) *SandboxHandler {
	return &SandboxHandler{svc: svc}
}

// Execute — POST /api/v1/sandbox/execute (recruiter-side sandbox; candidates
// run code through the WS code.run frame instead).
func (h *SandboxHandler) Execute(c *fiber.Ctx) error {
	actor, err := iamapi.RequireActor(c)
	if err != nil {
		return err
	}
	// CPU/LLM spend — members (viewers) are excluded, same gate as other
	// recruiter actions.
	if err := iamapp.Authorize(actor, iamdomain.RoleAdmin, iamdomain.RoleRecruiter, iamdomain.RoleInterviewer); err != nil {
		return httpapi.Error(c, err)
	}
	var req domain.ExecutionRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	result, err := h.svc.Execute(c.UserContext(), req)
	if err != nil {
		return httpapi.Error(c, err)
	}

	return httpapi.OK(c, result)
}

// Evaluate — POST /api/v1/sandbox/evaluate
func (h *SandboxHandler) Evaluate(c *fiber.Ctx) error {
	actor, err := iamapi.RequireActor(c)
	if err != nil {
		return err
	}
	if err := iamapp.Authorize(actor, iamdomain.RoleAdmin, iamdomain.RoleRecruiter, iamdomain.RoleInterviewer); err != nil {
		return httpapi.Error(c, err)
	}
	var req struct {
		Language domain.Language `json:"language"`
		Code     string          `json:"code"`
		Problem  string          `json:"problem"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	result, err := h.svc.EvaluateCode(c.UserContext(), req.Language, req.Code, req.Problem)
	if err != nil {
		return httpapi.Error(c, err)
	}

	return httpapi.OK(c, result)
}

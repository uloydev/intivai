package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	evalapp "github.com/intivai/backend/internal/evaluation/application"
	"github.com/intivai/backend/internal/iam/api"
	sharederr "github.com/intivai/backend/internal/shared/errors"
	"github.com/intivai/backend/internal/shared/httpapi"
)

// EvaluationHandler — recruiter-facing interview + report endpoints (P4a).
type EvaluationHandler struct {
	svc *evalapp.EvaluationService
}

func NewEvaluationHandler(svc *evalapp.EvaluationService) *EvaluationHandler {
	return &EvaluationHandler{svc: svc}
}

// GetInterview — GET /interviews/:id (full detail + evaluation).
func (h *EvaluationHandler) GetInterview(c *fiber.Ctx) error {
	actor, ok := api.Actor(c)
	if !ok {
		return httpapi.Error(c, sharederr.NewDomainError("UNAUTHORIZED", "unauthorized"))
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httpapi.Error(c, sharederr.NewDomainError("INVALID_INPUT", "invalid interview id"))
	}
	detail, err := h.svc.InterviewDetail(c.UserContext(), actor, id)
	if err != nil {
		return httpapi.Error(c, err)
	}
	return httpapi.OK(c, detail)
}

// GetCandidateReport — GET /candidates/:id/report (candidate + interviews).
func (h *EvaluationHandler) GetCandidateReport(c *fiber.Ctx) error {
	actor, ok := api.Actor(c)
	if !ok {
		return httpapi.Error(c, sharederr.NewDomainError("UNAUTHORIZED", "unauthorized"))
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httpapi.Error(c, sharederr.NewDomainError("INVALID_INPUT", "invalid candidate id"))
	}
	report, err := h.svc.CandidateReport(c.UserContext(), actor, id)
	if err != nil {
		return httpapi.Error(c, err)
	}
	return httpapi.OK(c, report)
}

// ListInterviews — GET /interviews (recruiter list view).
func (h *EvaluationHandler) ListInterviews(c *fiber.Ctx) error {
	actor, ok := api.Actor(c)
	if !ok {
		return httpapi.Error(c, sharederr.NewDomainError("UNAUTHORIZED", "unauthorized"))
	}
	list, err := h.svc.ListInterviews(c.UserContext(), actor)
	if err != nil {
		return httpapi.Error(c, err)
	}
	return httpapi.OK(c, list)
}

// GetInterviewPDF — GET /interviews/:id/report/pdf (download PDF report).
func (h *EvaluationHandler) GetInterviewPDF(c *fiber.Ctx) error {
	actor, ok := api.Actor(c)
	if !ok {
		return httpapi.Error(c, sharederr.NewDomainError("UNAUTHORIZED", "unauthorized"))
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httpapi.Error(c, sharederr.NewDomainError("BAD_REQUEST", "invalid interview id"))
	}
	pdfReader, err := h.svc.InterviewPDF(c.UserContext(), actor, id)
	if err != nil {
		return httpapi.Error(c, err)
	}

	c.Set("Content-Type", "application/pdf")
	c.Set("Content-Disposition", `attachment; filename="interview_report.pdf"`)
	return c.SendStream(pdfReader)
}

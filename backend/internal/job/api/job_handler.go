package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/intivai/backend/internal/iam/api"
	"github.com/intivai/backend/internal/job/application"
	"github.com/intivai/backend/internal/shared/httpapi"
)

func derefStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func derefSlice(p *[]string) []string {
	if p == nil {
		return nil
	}
	return *p
}

func derefInt(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

type JobHandler struct {
	svc *application.JobService
}

func NewJobHandler(svc *application.JobService) *JobHandler {
	return &JobHandler{svc: svc}
}

type jobRequest struct {
	Title             *string            `json:"title"`
	Description       *string            `json:"description"`
	Location          *string            `json:"location"`
	EmploymentType    *string            `json:"employment_type"`
	SalaryMin         *int               `json:"salary_min"`
	SalaryMax         *int               `json:"salary_max"`
	Currency          *string            `json:"currency"`
	RequiredSkills    *[]string          `json:"required_skills"`
	MinExperience     *int               `json:"min_experience"`
	Responsibilities  *[]string          `json:"responsibilities"`
	Requirements      *[]string          `json:"requirements"`
	NiceToHaves       *[]string          `json:"nice_to_haves"`
	Benefits          *[]string          `json:"benefits"`
	ScoringWeights    map[string]float64 `json:"scoring_weights"`
	MinScoreToProceed *float64           `json:"min_score_to_proceed"`
	Status            string             `json:"status"`
}

func (h *JobHandler) Create(c *fiber.Ctx) error {
	actor, err := api.RequireActor(c)
	if err != nil {
		return err
	}
	var req jobRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	result, err := h.svc.Create(c.UserContext(), actor, application.CreateJobCommand{
		Title:             derefStr(req.Title),
		Description:       derefStr(req.Description),
		Location:          derefStr(req.Location),
		EmploymentType:    derefStr(req.EmploymentType),
		SalaryMin:         req.SalaryMin,
		SalaryMax:         req.SalaryMax,
		Currency:          derefStr(req.Currency),
		RequiredSkills:    derefSlice(req.RequiredSkills),
		MinExperience:     derefInt(req.MinExperience),
		Responsibilities:  derefSlice(req.Responsibilities),
		Requirements:      derefSlice(req.Requirements),
		NiceToHaves:       derefSlice(req.NiceToHaves),
		Benefits:          derefSlice(req.Benefits),
		ScoringWeights:    req.ScoringWeights,
		MinScoreToProceed: req.MinScoreToProceed,
	})
	if err != nil {
		return httpapi.Error(c, err)
	}
	return httpapi.Created(c, result)
}

func (h *JobHandler) Update(c *fiber.Ctx) error {
	actor, err := api.RequireActor(c)
	if err != nil {
		return err
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid job id"})
	}
	var req jobRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	result, err := h.svc.Update(c.UserContext(), actor, application.UpdateJobCommand{
		JobID: id, Title: req.Title, Description: req.Description,
		Location: req.Location, EmploymentType: req.EmploymentType,
		SalaryMin: req.SalaryMin, SalaryMax: req.SalaryMax, Currency: req.Currency,
		RequiredSkills: req.RequiredSkills, MinExperience: req.MinExperience,
		Responsibilities: req.Responsibilities, Requirements: req.Requirements,
		NiceToHaves: req.NiceToHaves, Benefits: req.Benefits,
		ScoringWeights:    req.ScoringWeights,
		MinScoreToProceed: req.MinScoreToProceed, Status: req.Status,
	})
	if err != nil {
		return httpapi.Error(c, err)
	}
	return httpapi.OK(c, result)
}

func (h *JobHandler) Get(c *fiber.Ctx) error {
	actor, err := api.RequireActor(c)
	if err != nil {
		return err
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid job id"})
	}
	result, err := h.svc.Get(c.UserContext(), actor, id)
	if err != nil {
		return httpapi.Error(c, err)
	}
	return httpapi.OK(c, result)
}

func (h *JobHandler) List(c *fiber.Ctx) error {
	actor, err := api.RequireActor(c)
	if err != nil {
		return err
	}
	result, err := h.svc.List(c.UserContext(), actor)
	if err != nil {
		return httpapi.Error(c, err)
	}
	return httpapi.OK(c, result)
}

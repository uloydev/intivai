package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/intivai/backend/internal/iam/api"
	"github.com/intivai/backend/internal/integration/application"
	"github.com/intivai/backend/internal/shared/httpapi"
	"github.com/rs/zerolog"
)

type WebhookHandler struct {
	svc *application.WebhookService
	log zerolog.Logger
}

func NewWebhookHandler(svc *application.WebhookService, log zerolog.Logger) *WebhookHandler {
	return &WebhookHandler{svc: svc, log: log}
}

type webhookRequest struct {
	URL    *string  `json:"url"`
	Events []string `json:"events"`
	Secret *string  `json:"secret"`
	Active *bool    `json:"active"`
}

func (h *WebhookHandler) Create(c *fiber.Ctx) error {
	actor, err := api.RequireActor(c)
	if err != nil {
		return err
	}
	var req webhookRequest
	if err := c.BodyParser(&req); err != nil {
		return httpapi.Error(c, fiber.ErrBadRequest)
	}
	if req.URL == nil || req.Events == nil {
		return httpapi.Error(c, fiber.ErrBadRequest)
	}
	result, err := h.svc.Create(c.UserContext(), actor, application.CreateWebhookCommand{
		URL:    *req.URL,
		Events: req.Events,
		Secret: derefStr(req.Secret, ""),
	})
	if err != nil {
		return httpapi.Error(c, err)
	}
	return httpapi.Created(c, result)
}

func (h *WebhookHandler) List(c *fiber.Ctx) error {
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

func (h *WebhookHandler) Get(c *fiber.Ctx) error {
	actor, err := api.RequireActor(c)
	if err != nil {
		return err
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httpapi.Error(c, fiber.ErrBadRequest)
	}
	result, err := h.svc.Get(c.UserContext(), actor, id)
	if err != nil {
		return httpapi.Error(c, err)
	}
	return httpapi.OK(c, result)
}

func (h *WebhookHandler) Update(c *fiber.Ctx) error {
	actor, err := api.RequireActor(c)
	if err != nil {
		return err
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httpapi.Error(c, fiber.ErrBadRequest)
	}
	var req webhookRequest
	if err := c.BodyParser(&req); err != nil {
		return httpapi.Error(c, fiber.ErrBadRequest)
	}
	result, err := h.svc.Update(c.UserContext(), actor, id, application.UpdateWebhookCommand{
		URL:    req.URL,
		Events: req.Events,
		Secret: req.Secret,
		Active: req.Active,
	})
	if err != nil {
		return httpapi.Error(c, err)
	}
	return httpapi.OK(c, result)
}

func (h *WebhookHandler) Delete(c *fiber.Ctx) error {
	actor, err := api.RequireActor(c)
	if err != nil {
		return err
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httpapi.Error(c, fiber.ErrBadRequest)
	}
	if err := h.svc.Delete(c.UserContext(), actor, id); err != nil {
		return httpapi.Error(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *WebhookHandler) ListDeliveries(c *fiber.Ctx) error {
	actor, err := api.RequireActor(c)
	if err != nil {
		return err
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httpapi.Error(c, fiber.ErrBadRequest)
	}
	result, err := h.svc.ListDeliveries(c.UserContext(), actor, id)
	if err != nil {
		return httpapi.Error(c, err)
	}
	return httpapi.OK(c, result)
}

func derefStr(s *string, def string) string {
	if s != nil {
		return *s
	}
	return def
}

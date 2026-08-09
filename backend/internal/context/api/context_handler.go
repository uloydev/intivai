package api

import (
	"io"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	ctxapp "github.com/intivai/backend/internal/context/application"
	ctxdomain "github.com/intivai/backend/internal/context/domain"
	"github.com/intivai/backend/internal/iam/api"
	"github.com/intivai/backend/internal/shared/errors"
	"github.com/intivai/backend/internal/shared/httpapi"
)

type ContextHandler struct {
	svc *ctxapp.ContextService
}

func NewContextHandler(svc *ctxapp.ContextService) *ContextHandler {
	return &ContextHandler{svc: svc}
}

// orgFromPath validates :orgId matches the actor's org (tenant pinning).
func (h *ContextHandler) orgFromPath(c *fiber.Ctx) (uuid.UUID, error) {
	actor, err := api.RequireActor(c)
	if err != nil {
		return uuid.Nil, err
	}
	orgID, err := uuid.Parse(c.Params("orgId"))
	if err != nil || orgID != actor.OrgID {
		return uuid.Nil, errors.NewDomainError("FORBIDDEN", "org mismatch")
	}
	return orgID, nil
}

func (h *ContextHandler) UploadContext(c *fiber.Ctx) error {
	if _, err := h.orgFromPath(c); err != nil {
		return httpapi.Error(c, err)
	}
	actor, _ := api.Actor(c)

	contentType := ctxdomain.TypeText
	var content []byte
	switch {
	case strings.HasPrefix(c.Get("Content-Type"), "multipart/form-data"):
		contentType = ctxdomain.TypeFile
		fileHeader, err := c.FormFile("file")
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "file field required"})
		}
		file, err := fileHeader.Open()
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "cannot read file"})
		}
		defer file.Close()
		content, err = io.ReadAll(io.LimitReader(file, 128*1024))
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "cannot read file"})
		}
	default:
		var req struct {
			Content string `json:"content"`
		}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
		}
		content = []byte(req.Content)
	}

	result, err := h.svc.UploadContext(c.UserContext(), actor, contentType, content)
	if err != nil {
		return httpapi.Error(c, err)
	}
	return httpapi.Created(c, result)
}

func (h *ContextHandler) ListContexts(c *fiber.Ctx) error {
	if _, err := h.orgFromPath(c); err != nil {
		return httpapi.Error(c, err)
	}
	actor, _ := api.Actor(c)
	result, err := h.svc.ListContexts(c.UserContext(), actor)
	if err != nil {
		return httpapi.Error(c, err)
	}
	return httpapi.OK(c, result)
}

func (h *ContextHandler) SetPrompt(c *fiber.Ctx) error {
	if _, err := h.orgFromPath(c); err != nil {
		return httpapi.Error(c, err)
	}
	actor, _ := api.Actor(c)
	var req struct {
		SystemPrompt string `json:"system_prompt"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	result, err := h.svc.SetPrompt(c.UserContext(), actor, ctxapp.SetPromptCommand{SystemPrompt: req.SystemPrompt})
	if err != nil {
		return httpapi.Error(c, err)
	}
	return httpapi.OK(c, result)
}

func (h *ContextHandler) GetPrompt(c *fiber.Ctx) error {
	if _, err := h.orgFromPath(c); err != nil {
		return httpapi.Error(c, err)
	}
	actor, _ := api.Actor(c)
	result, err := h.svc.GetPrompt(c.UserContext(), actor)
	if err != nil {
		return httpapi.Error(c, err)
	}
	return httpapi.OK(c, result)
}

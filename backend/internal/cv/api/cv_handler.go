package api

import (
	"bytes"
	"io"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/intivai/backend/internal/cv/application"
	"github.com/intivai/backend/internal/iam/api"
	"github.com/intivai/backend/internal/shared/httpapi"
)

type CVHandler struct {
	svc         *application.CVService
	maxUploadMB int
}

func NewCVHandler(svc *application.CVService, maxUploadMB int) *CVHandler {
	return &CVHandler{svc: svc, maxUploadMB: maxUploadMB}
}

func (h *CVHandler) Upload(c *fiber.Ctx) error {
	actor, err := api.RequireActor(c)
	if err != nil {
		return err
	}
	fileHeader, err := c.FormFile("file")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "file field required"})
	}
	if fileHeader.Size == 0 {
		return c.Status(400).JSON(fiber.Map{"error": "uploaded file cannot be empty"})
	}
	if int(fileHeader.Size) > h.maxUploadMB*1024*1024 {
		return c.Status(413).JSON(fiber.Map{"error": "cv file too large"})
	}
	file, err := fileHeader.Open()
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "cannot read file"})
	}
	defer file.Close()
	data := make([]byte, fileHeader.Size)
	if _, err := io.ReadFull(file, data); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "cannot read file"})
	}
	if len(data) < 5 || !bytes.HasPrefix(data, []byte("%PDF-")) {
		return c.Status(400).JSON(fiber.Map{"error": "uploaded file must be a valid PDF format"})
	}

	result, err := h.svc.Upload(c.UserContext(), actor, c.FormValue("name"), c.FormValue("email"), data, fileHeader.Header.Get("Content-Type"))
	if err != nil {
		return httpapi.Error(c, err)
	}
	return httpapi.Created(c, result)
}

func (h *CVHandler) Get(c *fiber.Ctx) error {
	actor, err := api.RequireActor(c)
	if err != nil {
		return err
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid candidate id"})
	}
	result, err := h.svc.Get(c.UserContext(), actor, id)
	if err != nil {
		return httpapi.Error(c, err)
	}
	return httpapi.OK(c, result)
}

func (h *CVHandler) ReExtract(c *fiber.Ctx) error {
	actor, err := api.RequireActor(c)
	if err != nil {
		return err
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid candidate id"})
	}
	result, err := h.svc.ReExtract(c.UserContext(), actor, id)
	if err != nil {
		return httpapi.Error(c, err)
	}
	return httpapi.OK(c, result)
}

func (h *CVHandler) List(c *fiber.Ctx) error {
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

func (h *CVHandler) Delete(c *fiber.Ctx) error {
	actor, err := api.RequireActor(c)
	if err != nil {
		return err
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid candidate id"})
	}
	if err := h.svc.DeleteCandidate(c.UserContext(), actor, id); err != nil {
		return httpapi.Error(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

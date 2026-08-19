package api

import (
	"bytes"
	"encoding/json"
	"io"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/intivai/backend/internal/cv/application"
	"github.com/intivai/backend/internal/iam/api"
	sharederr "github.com/intivai/backend/internal/shared/errors"
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
		return httpapi.Error(c, sharederr.NewDomainError("BAD_REQUEST", "file field required"))
	}
	if fileHeader.Size == 0 {
		return httpapi.Error(c, sharederr.NewDomainError("BAD_REQUEST", "uploaded file cannot be empty"))
	}
	if int(fileHeader.Size) > h.maxUploadMB*1024*1024 {
		return httpapi.Error(c, sharederr.NewDomainError("PAYLOAD_TOO_LARGE", "cv file too large"))
	}
	file, err := fileHeader.Open()
	if err != nil {
		return httpapi.Error(c, sharederr.NewDomainError("BAD_REQUEST", "cannot read file"))
	}
	defer file.Close()
	data := make([]byte, fileHeader.Size)
	if _, err := io.ReadFull(file, data); err != nil {
		return httpapi.Error(c, sharederr.NewDomainError("BAD_REQUEST", "cannot read file"))
	}
	if len(data) < 5 || !bytes.HasPrefix(data, []byte("%PDF-")) {
		return httpapi.Error(c, sharederr.NewDomainError("BAD_REQUEST", "uploaded file must be a valid PDF format"))
	}

	result, err := h.svc.Upload(c.UserContext(), actor, c.FormValue("name"), c.FormValue("email"), data, fileHeader.Header.Get("Content-Type"))
	if err != nil {
		return httpapi.Error(c, err)
	}
	return httpapi.Created(c, result)
}

func (h *CVHandler) BulkUpload(c *fiber.Ctx) error {
	actor, err := api.RequireActor(c)
	if err != nil {
		return err
	}
	form, err := c.MultipartForm()
	if err != nil {
		return httpapi.Error(c, sharederr.NewDomainError("BAD_REQUEST", "failed to parse multipart form"))
	}
	fileHeaders, ok := form.File["files"]
	if !ok || len(fileHeaders) == 0 {
		return httpapi.Error(c, sharederr.NewDomainError("BAD_REQUEST", "files field required"))
	}

	var parsedFiles []application.BulkUploadFile
	for _, fh := range fileHeaders {
		if int(fh.Size) > h.maxUploadMB*1024*1024 {
			return httpapi.Error(c, sharederr.NewDomainError("PAYLOAD_TOO_LARGE", "one or more files too large"))
		}
		file, err := fh.Open()
		if err != nil {
			continue
		}
		data := make([]byte, fh.Size)
		if _, err := io.ReadFull(file, data); err == nil && len(data) >= 5 && bytes.HasPrefix(data, []byte("%PDF-")) {
			parsedFiles = append(parsedFiles, application.BulkUploadFile{
				Name:        fh.Filename,
				Data:        data,
				ContentType: fh.Header.Get("Content-Type"),
			})
		}
		file.Close()
	}

	if len(parsedFiles) == 0 {
		return httpapi.Error(c, sharederr.NewDomainError("BAD_REQUEST", "no valid pdf files found"))
	}

	result, err := h.svc.BulkUpload(c.UserContext(), actor, parsedFiles)
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
		return httpapi.Error(c, sharederr.NewDomainError("BAD_REQUEST", "invalid candidate id"))
	}
	result, err := h.svc.Get(c.UserContext(), actor, id)
	if err != nil {
		return httpapi.Error(c, err)
	}
	return httpapi.OK(c, result)
}

func (h *CVHandler) ReviewProfile(c *fiber.Ctx) error {
	token := c.Params("token")
	if token == "" {
		return httpapi.Error(c, sharederr.NewDomainError("BAD_REQUEST", "token is required"))
	}
	result, err := h.svc.ReviewProfile(c.Context(), token)
	if err != nil {
		return httpapi.Error(c, err)
	}
	return httpapi.OK(c, result)
}

func (h *CVHandler) ConfirmProfile(c *fiber.Ctx) error {
	token := c.Params("token")
	if token == "" {
		return httpapi.Error(c, sharederr.NewDomainError("BAD_REQUEST", "token is required"))
	}

	var req map[string]interface{}
	if err := c.BodyParser(&req); err != nil {
		return httpapi.Error(c, sharederr.NewDomainError("BAD_REQUEST", "invalid json payload"))
	}

	structuredData, err := json.Marshal(req)
	if err != nil {
		return httpapi.Error(c, sharederr.NewDomainError("BAD_REQUEST", "failed to serialize json payload"))
	}

	if err := h.svc.ConfirmProfile(c.Context(), token, structuredData); err != nil {
		return httpapi.Error(c, err)
	}

	return httpapi.OK(c, fiber.Map{"status": "confirmed"})
}

func (h *CVHandler) ReExtract(c *fiber.Ctx) error {
	actor, err := api.RequireActor(c)
	if err != nil {
		return err
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httpapi.Error(c, sharederr.NewDomainError("BAD_REQUEST", "invalid candidate id"))
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
		return httpapi.Error(c, sharederr.NewDomainError("BAD_REQUEST", "invalid candidate id"))
	}
	if err := h.svc.DeleteCandidate(c.UserContext(), actor, id); err != nil {
		return httpapi.Error(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

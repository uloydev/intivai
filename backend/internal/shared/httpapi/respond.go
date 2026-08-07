package httpapi

import (
	"errors"
	"net/http"

	"github.com/gofiber/fiber/v2"
	sharederr "github.com/intivai/backend/internal/shared/errors"
)

func Error(c *fiber.Ctx, err error) error {
	var nf *sharederr.NotFoundError
	if errors.As(err, &nf) {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": nf.Error()})
	}
	var de *sharederr.DomainError
	if errors.As(err, &de) {
		status := http.StatusBadRequest
		switch de.Code {
		case "AUTH_FAILED", "UNAUTHORIZED":
			status = http.StatusUnauthorized
		case "FORBIDDEN":
			status = http.StatusForbidden
		}
		return c.Status(status).JSON(fiber.Map{"error": de.Message, "code": de.Code})
	}
	return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "internal server error"})
}

func OK(c *fiber.Ctx, data any) error {
	return c.JSON(fiber.Map{"data": data})
}

func Created(c *fiber.Ctx, data any) error {
	return c.Status(http.StatusCreated).JSON(fiber.Map{"data": data})
}

package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgconn"

	sharederr "github.com/intivai/backend/internal/shared/errors"
	"github.com/intivai/backend/pkg/db"
)

// A repo failure that reaches the handler through db.WrapError (as every
// repository does) must render as a generic 500 when it is an internal fault
// — not as 400 carrying driver text like "context canceled" (finding D21).
func TestErrorRendersWrappedInternalAs500(t *testing.T) {
	app := fiber.New()
	app.Post("/cvs", func(c *fiber.Ctx) error {
		// Exact value shape repositories emit when a tenant-tx context dies
		// mid-request (gorm wraps the driver error; WrapError classifies).
		repoErr := db.WrapError(fmt.Errorf("update candidates: %w", context.Canceled))
		return Error(c, repoErr)
	})

	resp, err := app.Test(httptest.NewRequest(http.MethodPost, "/cvs", nil), -1)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", resp.StatusCode)
	}
	var body map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&body)
	if body["error"] != "internal server error" {
		t.Fatalf("body = %v, want generic internal server error", body)
	}
}

func TestErrorStillMapsUniqueViolationTo400Sentinel(t *testing.T) {
	app := fiber.New()
	app.Post("/dup", func(c *fiber.Ctx) error {
		pgErr := &pgconn.PgError{Code: "23505", ConstraintName: "jobs_slug_key"}
		return Error(c, db.WrapError(pgErr))
	})

	resp, err := app.Test(httptest.NewRequest(http.MethodPost, "/dup", nil), -1)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
	var body map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&body)
	if body["code"] != "ALREADY_EXISTS" {
		t.Fatalf("code = %v, want ALREADY_EXISTS", body["code"])
	}
	if body["error"] == "context canceled" {
		t.Fatal("leaked internal text")
	}
}

func TestErrorKnownShapesUnchanged(t *testing.T) {
	app := fiber.New()
	app.Get("/nf", func(c *fiber.Ctx) error {
		return Error(c, sharederr.NewNotFoundError("candidate", "abc"))
	})
	app.Get("/dom", func(c *fiber.Ctx) error {
		return Error(c, sharederr.NewDomainError("FORBIDDEN", "nope"))
	})
	app.Get("/raw", func(c *fiber.Ctx) error {
		return Error(c, errors.New("totally unexpected"))
	})

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/nf", nil), -1)
	if err != nil {
		t.Fatal(err)
	}
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("not-found status = %d, want 404", resp.StatusCode)
	}
	respDom, err := app.Test(httptest.NewRequest(http.MethodGet, "/dom", nil), -1)
	if err != nil {
		t.Fatal(err)
	}
	if respDom != nil && respDom.Body != nil {
		defer respDom.Body.Close()
	}
	if respDom.StatusCode != http.StatusForbidden {
		t.Fatalf("forbidden status = %d, want 403", respDom.StatusCode)
	}
	respRaw, err := app.Test(httptest.NewRequest(http.MethodGet, "/raw", nil), -1)
	if err != nil {
		t.Fatal(err)
	}
	if respRaw != nil && respRaw.Body != nil {
		defer respRaw.Body.Close()
	}
	if respRaw.StatusCode != http.StatusInternalServerError {
		t.Fatalf("raw error status = %d, want 500", respRaw.StatusCode)
	}
}

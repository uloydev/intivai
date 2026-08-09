package api

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/intivai/backend/internal/iam/application"
)

type stubTokens struct {
	claims *application.Claims
	err    error
}

func (s stubTokens) Issue(_ uuid.UUID, _ uuid.UUID, _, _ string, _ time.Duration, _ map[string]any) (string, error) {
	return "", nil
}
func (s stubTokens) Parse(_ string) (*application.Claims, error) {
	return s.claims, s.err
}

func doAuthedRequest(app *fiber.App) int {
	r := httptest.NewRequest(http.MethodGet, "/protected", nil)
	r.Header.Set("Authorization", "Bearer x")
	resp, err := app.Test(r, -1)
	if err != nil {
		return -1
	}
	return resp.StatusCode
}

func TestAuthMiddlewareRejectsNonAuthTokenType(t *testing.T) {
	app := fiber.New()
	app.Get("/protected", AuthMiddleware(stubTokens{claims: &application.Claims{
		Subject: uuid.New(), OrgID: uuid.New(), Role: "admin", Type: application.TokenTypeWSTicket,
	}}), func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) })

	if code := doAuthedRequest(app); code != fiber.StatusUnauthorized {
		t.Fatalf("ws_ticket accepted on API route: status %d, want 401", code)
	}
}

func TestAuthMiddlewareRejectsInvalidToken(t *testing.T) {
	app := fiber.New()
	app.Get("/protected", AuthMiddleware(stubTokens{err: errTestToken}), func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	if code := doAuthedRequest(app); code != fiber.StatusUnauthorized {
		t.Fatalf("invalid token accepted: status %d, want 401", code)
	}
}

func TestAuthMiddlewareAcceptsAuthToken(t *testing.T) {
	app := fiber.New()
	app.Get("/protected", AuthMiddleware(stubTokens{claims: &application.Claims{
		Subject: uuid.New(), OrgID: uuid.New(), Role: "admin", Type: application.TokenTypeAuth,
	}}), func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) })

	if code := doAuthedRequest(app); code != fiber.StatusOK {
		t.Fatalf("auth token rejected: status %d, want 200", code)
	}
}

var errTestToken = errors.New("invalid token")

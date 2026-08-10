package main

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

// Fiber regression guard: Group("", handlers...) registers app.Use("/") —
// GLOBAL middleware for every route registered AFTER it. Public (candidate)
// routes must be registered BEFORE any empty-prefix middleware group, or
// they silently inherit auth/tenant middleware.
func TestPublicRoutesNotAffectedByLaterAuthGroup(t *testing.T) {
	app := fiber.New()
	v1 := app.Group("/api/v1")

	// Public routes registered BEFORE the middleware group.
	v1.Post("/candidate/interviews/:id/ticket", func(c *fiber.Ctx) error { return c.SendStatus(200) })
	v1.Get("/candidate/interviews/:id/chat", func(c *fiber.Ctx) error { return c.SendStatus(200) })

	// Empty-prefix group with middleware — makes authMW global from here on.
	_ = v1.Group("", func(c *fiber.Ctx) error {
		return c.Status(401).JSON(fiber.Map{"error": "missing bearer token"})
	})

	cases := []struct {
		method, path string
	}{
		{"POST", "/api/v1/candidate/interviews/abc/ticket"},
		{"GET", "/api/v1/candidate/interviews/abc/chat"},
	}
	for _, c := range cases {
		req := httptest.NewRequest(c.method, c.path, nil)
		resp, _ := app.Test(req, -1)
		if resp.StatusCode != 200 {
			t.Fatalf("%s %s got %d — auth middleware leaked onto public route", c.method, c.path, resp.StatusCode)
		}
	}
}

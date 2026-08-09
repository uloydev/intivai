package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/intivai/backend/pkg/db"
)

// TenantMiddleware attaches org_id to the request context so RLS policies
// resolve via current_setting('app.org_id'). Must run AFTER AuthMiddleware.
func TenantMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		actor, ok := Actor(c)
		if !ok {
			return c.Next()
		}
		c.SetUserContext(db.WithTenant(c.UserContext(), actor.OrgID.String()))
		return c.Next()
	}
}

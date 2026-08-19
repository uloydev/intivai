package api

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/intivai/backend/internal/iam/application"
	sharederr "github.com/intivai/backend/internal/shared/errors"
	"github.com/intivai/backend/internal/shared/httpapi"
)

const authCtxKey = "auth"

// AuthMiddleware parses the Bearer JWT and stores AuthContext in locals.
// Rejects unauthenticated requests (401) and non-auth tokens (ws_tickets
// are candidate-only credentials, never accepted on API routes).
func AuthMiddleware(tokens application.TokenProvider) fiber.Handler {
	return func(c *fiber.Ctx) error {
		header := c.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			return httpapi.Error(c, sharederr.NewDomainError("UNAUTHORIZED", "missing bearer token"))
		}
		claims, err := tokens.Parse(strings.TrimPrefix(header, "Bearer "))
		if err != nil {
			return httpapi.Error(c, sharederr.NewDomainError("UNAUTHORIZED", "invalid or expired token"))
		}
		if claims.Type != application.TokenTypeAuth {
			return httpapi.Error(c, sharederr.NewDomainError("UNAUTHORIZED", "invalid token type"))
		}
		actor := application.AuthContext{
			UserID: claims.Subject,
			OrgID:  claims.OrgID,
			Role:   claims.Role,
		}
		c.Locals(authCtxKey, actor)
		return c.Next()
	}
}

// OptionalAuthMiddleware lets endpoints work for authenticated AND anonymous
// callers (candidate token flow). Stores nil actor when unauthenticated.
// Only auth-type tokens are honored.
func OptionalAuthMiddleware(tokens application.TokenProvider) fiber.Handler {
	return func(c *fiber.Ctx) error {
		header := c.Get("Authorization")
		if strings.HasPrefix(header, "Bearer ") {
			if claims, err := tokens.Parse(strings.TrimPrefix(header, "Bearer ")); err == nil && claims.Type == application.TokenTypeAuth {
				c.Locals(authCtxKey, application.AuthContext{
					UserID: claims.Subject,
					OrgID:  claims.OrgID,
					Role:   claims.Role,
				})
			}
		}
		return c.Next()
	}
}

func Actor(c *fiber.Ctx) (application.AuthContext, bool) {
	v, ok := c.Locals(authCtxKey).(application.AuthContext)
	return v, ok
}

func RequireActor(c *fiber.Ctx) (application.AuthContext, error) {
	actor, ok := Actor(c)
	if !ok {
		return actor, sharederr.NewDomainError("UNAUTHORIZED", "unauthenticated")
	}
	return actor, nil
}

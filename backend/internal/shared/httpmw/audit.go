package httpmw

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// RequestID assigns a request id and exposes it via the X-Request-Id header.
func RequestID(log zerolog.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := c.Get("X-Request-Id")
		if id == "" {
			id = uuid.NewString()
		}
		c.Set("X-Request-Id", id)
		c.Locals("request_id", id)
		logger := log.With().Str("request_id", id).Logger()
		c.Locals("logger", logger)
		return c.Next()
	}
}

// Audit logs every request with identity + tenant context.
// Post-MVP: persist to audit_logs table (action-level audit is use-case driven).
func Audit(log zerolog.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		logger, _ := c.Locals("logger").(zerolog.Logger)
		if logger.GetLevel() == zerolog.NoLevel {
			logger = log
		}
		logger.Info().
			Str("method", c.Method()).
			Str("path", c.Path()).
			Int("status", c.Response().StatusCode()).
			Dur("latency_ms", time.Since(start)).
			Str("ip", c.IP()).
			Msg("request")
		return err
	}
}

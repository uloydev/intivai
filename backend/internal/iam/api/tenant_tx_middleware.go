package api

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/intivai/backend/pkg/db"
	"gorm.io/gorm"
)

// TenantTxMiddleware opens a per-request transaction, configures the tenant
// (app.org_id) on it so RLS policies resolve, and attaches it to the request
// context. Repos resolve it via db.TxFrom — RLS-scoped tables must never be
// touched outside one. Must run AFTER AuthMiddleware.
func TenantTxMiddleware(pool *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		actor, ok := Actor(c)
		if !ok {
			return c.Next()
		}

		ctx := c.UserContext()
		tx := pool.WithContext(ctx).Begin()
		if tx.Error != nil {
			return tx.Error
		}
		defer tx.Rollback()

		if err := db.SetTenant(ctx, tx, actor.OrgID.String()); err != nil {
			return err
		}

		c.SetUserContext(db.WithTx(ctx, tx))

		if err := c.Next(); err != nil {
			return err
		}

		// Handler already wrote an error response (>=400): the transaction is
		// aborted in Postgres, so roll back via defer instead of committing.
		if c.Response().StatusCode() >= fiber.StatusBadRequest {
			return nil
		}

		commitCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return tx.WithContext(commitCtx).Commit().Error
	}
}

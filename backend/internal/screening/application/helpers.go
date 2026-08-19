package application

import (
	"context"

	scrdomain "github.com/intivai/backend/internal/screening/domain"
	"gorm.io/gorm"
)

// CreateApplicationWithRecovery — insert an application inside a tenant
// transaction, recovering from a concurrent-create unique violation: a failed
// statement aborts the whole tx (25P02), so roll back to the savepoint and
// reload the winning row. Returns the surviving application and whether this
// call created it.
func CreateApplicationWithRecovery(
	ctx context.Context,
	tx *gorm.DB,
	appRepo scrdomain.ApplicationRepository,
	app *scrdomain.Application,
) (*scrdomain.Application, bool, error) {
	if err := tx.SavePoint("create_app").Error; err != nil {
		return nil, false, err
	}
	if err := appRepo.Create(ctx, app); err != nil {
		if !scrdomain.IsExists(err) {
			return nil, false, err
		}
		// Concurrent create — roll back to the savepoint, reload the winner.
		if rerr := tx.RollbackTo("create_app").Error; rerr != nil {
			return nil, false, rerr
		}
		winning, err := appRepo.GetByCandidateJob(ctx, app.OrgID, app.CandidateID, app.JobID)
		if err != nil {
			return nil, false, err
		}
		return winning, false, nil
	}
	return app, true, nil
}

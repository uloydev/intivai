package persistence

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	jobdomain "github.com/intivai/backend/internal/job/domain"
	"github.com/intivai/backend/pkg/db"
	"gorm.io/gorm"
)

// PostgresCandidateContextRepo implements jobdomain.CandidateContextRepository.
// All reads/writes run inside a tenant transaction (db.TxFrom) so FORCED RLS
// applies (027).
type PostgresCandidateContextRepo struct {
	pool *gorm.DB
}

func NewPostgresCandidateContextRepo(pool *gorm.DB) *PostgresCandidateContextRepo {
	return &PostgresCandidateContextRepo{pool: pool}
}

func (r *PostgresCandidateContextRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := db.TxFrom(ctx)
	if !ok {
		return nil, db.ErrNoTx
	}
	return tx, nil
}

const candidateContextColumns = `job_id, org_id, content, version, updated_at`

// Upsert writes the context for a job. The version is bumped on every save
// (ON CONFLICT .. DO UPDATE sets version = version + 1) so each edit is
// observable and pinnable at connect time. RETURNING version/updated_at (J12)
// keeps the caller's aggregate in sync with the row the DB just stamped —
// a pre-write copy would report version 1 after an update bumped it to N.
func (r *PostgresCandidateContextRepo) Upsert(ctx context.Context, c *jobdomain.CandidateContext) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	return tx.WithContext(ctx).Raw(
		`INSERT INTO job_candidate_contexts (job_id, org_id, content, version, updated_at)
		 VALUES ($1, $2, $3, 1, NOW())
		 ON CONFLICT (job_id) DO UPDATE SET
		   org_id = EXCLUDED.org_id,
		   content = EXCLUDED.content,
		   version = job_candidate_contexts.version + 1,
		   updated_at = NOW()
		 RETURNING version, updated_at`,
		c.JobID, c.OrgID, c.Content).Row().Scan(&c.Version, &c.UpdatedAt)
}

func (r *PostgresCandidateContextRepo) GetByJobID(ctx context.Context, jobID uuid.UUID) (*jobdomain.CandidateContext, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	row := tx.Raw(
		`SELECT `+candidateContextColumns+` FROM job_candidate_contexts WHERE job_id = $1`, jobID).Row()
	return scanCandidateContext(row)
}

func scanCandidateContext(row rowScanner) (*jobdomain.CandidateContext, error) {
	var (
		c         jobdomain.CandidateContext
		updatedAt time.Time
	)
	err := row.Scan(&c.JobID, &c.OrgID, &c.Content, &c.Version, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, jobdomain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	c.UpdatedAt = updatedAt
	return &c, nil
}

package persistence

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	cvdomain "github.com/intivai/backend/internal/cv/domain"
	"github.com/intivai/backend/pkg/db"
	"gorm.io/gorm"
)

type PostgresCandidateRepo struct {
	pool *gorm.DB
}

func NewPostgresCandidateRepo(pool *gorm.DB) *PostgresCandidateRepo {
	return &PostgresCandidateRepo{pool: pool}
}

func (r *PostgresCandidateRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := db.TxFrom(ctx)
	if !ok {
		return nil, db.ErrNoTx
	}
	return tx, nil
}

func (r *PostgresCandidateRepo) Create(ctx context.Context, c *cvdomain.Candidate) error {
	q, err := r.tx(ctx)
	if err != nil {
		return err
	}
	err = q.WithContext(ctx).Exec(
		`INSERT INTO candidates (id, org_id, name, email, cv_path, status, batch_id, review_token, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		c.ID, c.OrgID, c.Name, c.Email, c.CVPath, c.Status, c.BatchID, c.ReviewToken, c.CreatedAt).Error
	return db.WrapError(err)
}

func (r *PostgresCandidateRepo) GetByID(ctx context.Context, id uuid.UUID) (*cvdomain.Candidate, error) {
	q, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	row := q.Raw(
		`SELECT id, org_id, name, email, cv_path, cv_raw_text, cv_structured, cv_ocr_method, status, error_message, batch_id, review_token, created_at
		 FROM candidates WHERE id = $1`, id).Row()
	return scanCandidate(row)
}

// GetByReviewToken — public review flow (no tenant context exists): the
// cross-org token lookup runs through the SECURITY DEFINER function.
func (r *PostgresCandidateRepo) GetByReviewToken(ctx context.Context, token string) (*cvdomain.Candidate, error) {
	row := r.pool.WithContext(ctx).Raw(
		`SELECT id, org_id, name, email, cv_path, cv_raw_text, cv_structured, cv_ocr_method, status, error_message, batch_id, review_token, created_at
		 FROM candidate_by_review_token($1)`, token).Row()
	return scanCandidate(row)
}

// ConfirmReview — atomically confirm the extracted profile (SECURITY DEFINER);
// returns the candidate's org + id, or uuid.Nil when the token is invalid or
// the candidate is no longer pending review.
func (r *PostgresCandidateRepo) ConfirmReview(ctx context.Context, token string, structured []byte) (uuid.UUID, uuid.UUID, error) {
	var orgID, candID uuid.UUID
	row := r.pool.WithContext(ctx).Raw(
		`SELECT org_id, candidate_id FROM candidate_confirm_review($1, $2)`, token, string(structured)).Row()
	err := row.Scan(&orgID, &candID)
	if errors.Is(err, sql.ErrNoRows) {
		return uuid.Nil, uuid.Nil, nil
	}
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	return orgID, candID, nil
}

func (r *PostgresCandidateRepo) List(ctx context.Context, orgID uuid.UUID) ([]*cvdomain.Candidate, error) {
	q, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := q.Raw(
		`SELECT id, org_id, name, email, cv_path, cv_raw_text, cv_structured, cv_ocr_method, status, error_message, batch_id, review_token, created_at
		 FROM candidates WHERE org_id = $1 ORDER BY created_at DESC`, orgID).Rows()
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []*cvdomain.Candidate{}
	for rows.Next() {
		c, err := scanCandidate(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *PostgresCandidateRepo) Update(ctx context.Context, c *cvdomain.Candidate) error {
	q, err := r.tx(ctx)
	if err != nil {
		return err
	}
	err = q.WithContext(ctx).Exec(
		`UPDATE candidates SET name = $1, email = $2, cv_path = $3, cv_raw_text = $4,
		 cv_structured = $5, cv_ocr_method = $6, status = $7, error_message = $8, batch_id = $9, review_token = $10, updated_at = NOW()
		 WHERE id = $11`,
		c.Name, c.Email, c.CVPath, c.CVRawText, c.CVStructured, c.CVOCRMethod, c.Status, c.ErrorMessage, c.BatchID, c.ReviewToken, c.ID).Error
	return db.WrapError(err)
}

func (r *PostgresCandidateRepo) Delete(ctx context.Context, id uuid.UUID) error {
	q, err := r.tx(ctx)
	if err != nil {
		return err
	}
	err = q.WithContext(ctx).Exec(`DELETE FROM candidates WHERE id = $1`, id).Error
	return db.WrapError(err)
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanCandidate(row rowScanner) (*cvdomain.Candidate, error) {
	var (
		c          cvdomain.Candidate
		path       *string
		raw        *string
		structured []byte
		errMsg     *string
		ocrMethod  *string
	)
	err := row.Scan(&c.ID, &c.OrgID, &c.Name, &c.Email, &path, &raw, &structured, &ocrMethod, &c.Status, &errMsg, &c.BatchID, &c.ReviewToken, &c.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, cvdomain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if path != nil {
		c.CVPath = *path
	}
	if raw != nil {
		c.CVRawText = *raw
	}
	if errMsg != nil {
		c.ErrorMessage = *errMsg
	}
	if ocrMethod != nil {
		c.CVOCRMethod = *ocrMethod
	}
	if len(structured) > 0 {
		c.CVStructured = structured
	}
	return &c, nil
}

// ListByIDs — batched candidate fetch for list enrichment (RLS-scoped).
func (r *PostgresCandidateRepo) ListByIDs(ctx context.Context, orgID uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]*cvdomain.Candidate, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return map[uuid.UUID]*cvdomain.Candidate{}, nil
	}
	rows, err := tx.WithContext(ctx).Raw(
		`SELECT id, org_id, name, email, COALESCE(cv_path, ''), status, created_at FROM candidates
		 WHERE org_id = $1 AND id = ANY($2)`, orgID, ids).Rows()
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := map[uuid.UUID]*cvdomain.Candidate{}
	for rows.Next() {
		var c cvdomain.Candidate
		if err := rows.Scan(&c.ID, &c.OrgID, &c.Name, &c.Email, &c.CVPath, &c.Status, &c.CreatedAt); err != nil {
			return nil, err
		}
		out[c.ID] = &c
	}
	return out, rows.Err()
}

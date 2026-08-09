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

func (r *PostgresCandidateRepo) q(ctx context.Context) (*gorm.DB, error) {
	tx, ok := db.TxFrom(ctx)
	if !ok {
		return nil, db.ErrNoTx
	}
	return tx, nil
}

func (r *PostgresCandidateRepo) Create(ctx context.Context, c *cvdomain.Candidate) error {
	q, err := r.q(ctx)
	if err != nil {
		return err
	}
	return q.WithContext(ctx).Exec(
		`INSERT INTO candidates (id, org_id, name, email, cv_path, status, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		c.ID, c.OrgID, c.Name, c.Email, c.CVPath, c.Status, c.CreatedAt).Error
}

func (r *PostgresCandidateRepo) GetByID(ctx context.Context, id uuid.UUID) (*cvdomain.Candidate, error) {
	q, err := r.q(ctx)
	if err != nil {
		return nil, err
	}
	row := q.Raw(
		`SELECT id, org_id, name, email, cv_path, cv_raw_text, cv_structured, cv_ocr_method, status, error_message, created_at
		 FROM candidates WHERE id = $1`, id).Row()
	return scanCandidate(row)
}

func (r *PostgresCandidateRepo) List(ctx context.Context, orgID uuid.UUID) ([]*cvdomain.Candidate, error) {
	q, err := r.q(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := q.Raw(
		`SELECT id, org_id, name, email, cv_path, cv_raw_text, cv_structured, cv_ocr_method, status, error_message, created_at
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
	q, err := r.q(ctx)
	if err != nil {
		return err
	}
	return q.WithContext(ctx).Exec(
		`UPDATE candidates SET name = $1, email = $2, cv_path = $3, cv_raw_text = $4,
		 cv_structured = $5, cv_ocr_method = $6, status = $7, error_message = $8, updated_at = NOW()
		 WHERE id = $9`,
		c.Name, c.Email, c.CVPath, c.CVRawText, c.CVStructured, c.CVOCRMethod, c.Status, c.ErrorMessage, c.ID).Error
}

func (r *PostgresCandidateRepo) Delete(ctx context.Context, id uuid.UUID) error {
	q, err := r.q(ctx)
	if err != nil {
		return err
	}
	return q.WithContext(ctx).Exec(`DELETE FROM candidates WHERE id = $1`, id).Error
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
	err := row.Scan(&c.ID, &c.OrgID, &c.Name, &c.Email, &path, &raw, &structured, &ocrMethod, &c.Status, &errMsg, &c.CreatedAt)
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

package persistence

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	ctxdomain "github.com/intivai/backend/internal/context/domain"
	"github.com/intivai/backend/pkg/db"
	"gorm.io/gorm"
)

type PostgresContextRepo struct {
	pool *gorm.DB
}

func NewPostgresContextRepo(pool *gorm.DB) *PostgresContextRepo {
	return &PostgresContextRepo{pool: pool}
}

func (r *PostgresContextRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := db.TxFrom(ctx)
	if !ok {
		return nil, db.ErrNoTx
	}
	return tx, nil
}

func (r *PostgresContextRepo) CreateContext(ctx context.Context, cc *ctxdomain.CompanyContext) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	return tx.WithContext(ctx).Exec(
		`INSERT INTO company_contexts (id, org_id, type, content_hash, version, storage_path, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		cc.ID, cc.OrgID, cc.Type, cc.ContentHash, cc.Version, cc.StoragePath, cc.CreatedAt).Error
}

func (r *PostgresContextRepo) GetContextByHash(ctx context.Context, orgID uuid.UUID, hash string) (*ctxdomain.CompanyContext, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	row := tx.Raw(
		`SELECT id, org_id, type, content_hash, version, storage_path, created_at
		 FROM company_contexts WHERE org_id = $1 AND content_hash = $2 ORDER BY version DESC LIMIT 1`,
		orgID, hash).Row()
	return scanContext(row)
}

func (r *PostgresContextRepo) GetContextByID(ctx context.Context, id uuid.UUID) (*ctxdomain.CompanyContext, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	row := tx.Raw(
		`SELECT id, org_id, type, content_hash, version, storage_path, created_at
		 FROM company_contexts WHERE id = $1`, id).Row()
	return scanContext(row)
}

func (r *PostgresContextRepo) ListContexts(ctx context.Context, orgID uuid.UUID) ([]*ctxdomain.CompanyContext, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := tx.Raw(
		`SELECT id, org_id, type, content_hash, version, storage_path, created_at
		 FROM company_contexts WHERE org_id = $1 ORDER BY version DESC`, orgID).Rows()
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []*ctxdomain.CompanyContext{}
	for rows.Next() {
		cc, err := scanContext(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, cc)
	}
	return out, rows.Err()
}

func (r *PostgresContextRepo) SetPrompt(ctx context.Context, p *ctxdomain.TenantPrompt) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	return tx.WithContext(ctx).Exec(
		`INSERT INTO tenant_prompts (id, org_id, system_prompt, version, created_at)
		 VALUES ($1, $2, $3, $4, NOW())`,
		uuid.NewString(), p.OrgID, p.SystemPrompt, p.Version).Error
}

func (r *PostgresContextRepo) GetLatestPrompt(ctx context.Context, orgID uuid.UUID) (*ctxdomain.TenantPrompt, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	row := tx.Raw(
		`SELECT org_id, system_prompt, version, created_at
		 FROM tenant_prompts WHERE org_id = $1 ORDER BY version DESC LIMIT 1`, orgID).Row()
	var p ctxdomain.TenantPrompt
	err = row.Scan(&p.OrgID, &p.SystemPrompt, &p.Version, &p.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ctxdomain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanContext(row rowScanner) (*ctxdomain.CompanyContext, error) {
	var (
		cc          ctxdomain.CompanyContext
		storagePath *string
	)
	err := row.Scan(&cc.ID, &cc.OrgID, &cc.Type, &cc.ContentHash, &cc.Version, &storagePath, &cc.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ctxdomain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if storagePath != nil {
		cc.StoragePath = *storagePath
	}
	return &cc, nil
}

func (r *PostgresContextRepo) DeleteContext(ctx context.Context, orgID, id uuid.UUID) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	return tx.WithContext(ctx).Exec(
		`DELETE FROM company_contexts WHERE id = $1 AND org_id = $2`, id, orgID).Error
}

package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	iamdomain "github.com/intivai/backend/internal/iam/domain"
	"github.com/intivai/backend/pkg/db"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

type PostgresIAMRepo struct {
	pool *gorm.DB
}

func NewPostgresIAMRepo(pool *gorm.DB) *PostgresIAMRepo {
	return &PostgresIAMRepo{pool: pool}
}

// tx REQUIRES a tenant transaction — RLS-scoped tables must never be
// touched outside one, otherwise queries silently return zero rows.
func (r *PostgresIAMRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := db.TxFrom(ctx)
	if !ok {
		return nil, db.ErrNoTx
	}
	return tx, nil
}

func (r *PostgresIAMRepo) CreateOrg(ctx context.Context, org *iamdomain.Org) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	var weights []byte
	if org.ScoringWeights != nil {
		var err error
		weights, err = org.MarshalScoringWeights()
		if err != nil {
			return err
		}
	}
	err = tx.WithContext(ctx).Exec(
		`INSERT INTO orgs (id, name, slug, plan, scoring_weights, min_score_to_proceed, candidate_qa_limit, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		org.ID, org.Name, org.Slug, org.Plan, weights, org.MinScoreToProceed, org.CandidateQALimit, org.CreatedAt).Error
	return mapDuplicate(err, "orgs_slug_key", iamdomain.ErrDuplicateSlug)
}

func (r *PostgresIAMRepo) GetOrg(ctx context.Context, id uuid.UUID) (*iamdomain.Org, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	row := tx.Raw(
		`SELECT id, name, slug, plan, scoring_weights, min_score_to_proceed, candidate_qa_limit, created_at FROM orgs WHERE id = $1`, id).Row()
	return scanOrg(row)
}

func (r *PostgresIAMRepo) GetOrgBySlug(ctx context.Context, slug string) (*iamdomain.Org, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	row := tx.Raw(
		`SELECT id, name, slug, plan, scoring_weights, min_score_to_proceed, candidate_qa_limit, created_at FROM orgs WHERE slug = $1`, slug).Row()
	return scanOrg(row)
}

// UpdateOrgCandidateQALimit sets the per-tenant candidate Q&A cap (B4/D3).
// The orgs RLS policy scopes the UPDATE to app.org_id; a zero-row result means
// the org is unknown to the caller's tenant — mapped to ErrNotFound. The orgs
// table has no updated_at column (migration 004), so none is touched here.
func (r *PostgresIAMRepo) UpdateOrgCandidateQALimit(ctx context.Context, orgID uuid.UUID, limit int) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	res := tx.WithContext(ctx).Exec(
		`UPDATE orgs SET candidate_qa_limit = $2 WHERE id = $1`, orgID, limit)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return iamdomain.ErrNotFound
	}
	return nil
}

type orgScanner interface {
	Scan(dest ...any) error
}

func scanOrg(row orgScanner) (*iamdomain.Org, error) {
	var (
		org      iamdomain.Org
		weights  []byte
		minScore *float64
		qaLimit  *int
	)
	err := row.Scan(&org.ID, &org.Name, &org.Slug, &org.Plan, &weights, &minScore, &qaLimit, &org.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, iamdomain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	org.MinScoreToProceed = minScore
	org.CandidateQALimit = qaLimit
	if len(weights) > 0 {
		if err := json.Unmarshal(weights, &org.ScoringWeights); err != nil {
			return nil, fmt.Errorf("decode scoring weights: %w", err)
		}
	}
	return &org, nil
}

func (r *PostgresIAMRepo) CreateUser(ctx context.Context, user *iamdomain.User) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	err = tx.WithContext(ctx).Exec(
		`INSERT INTO users (id, org_id, email, role, password_hash, auth_provider, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		user.ID, user.OrgID, user.Email, string(user.Role), nilString(user.PasswordHash), user.AuthProvider, user.CreatedAt).Error
	return mapDuplicate(err, "users_org_id_email_key", iamdomain.ErrDuplicateEmail)
}

func (r *PostgresIAMRepo) GetUserByID(ctx context.Context, id uuid.UUID) (*iamdomain.User, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	row := tx.Raw(
		`SELECT id, org_id, email, role, password_hash, auth_provider, created_at FROM users WHERE id = $1`, id).Row()
	return scanUser(row)
}

func (r *PostgresIAMRepo) GetUserByEmail(ctx context.Context, orgID uuid.UUID, email string) (*iamdomain.User, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	row := tx.Raw(
		`SELECT id, org_id, email, role, password_hash, auth_provider, created_at FROM users WHERE org_id = $1 AND email = $2`,
		orgID, email).Row()
	return scanUser(row)
}

func (r *PostgresIAMRepo) ListUsers(ctx context.Context, orgID uuid.UUID) ([]*iamdomain.User, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := tx.Raw(
		`SELECT id, org_id, email, role, password_hash, auth_provider, created_at FROM users WHERE org_id = $1 ORDER BY created_at`,
		orgID).Rows()
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	users := []*iamdomain.User{}
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// FindLoginIdentity uses the security-definer function login_lookup — it must
// work WITHOUT a tenant context (pre-auth).
func (r *PostgresIAMRepo) FindLoginIdentity(ctx context.Context, orgSlug, email string) (*iamdomain.LoginIdentity, error) {
	row := r.pool.WithContext(ctx).Raw(
		`SELECT org_id, user_id, email, password_hash, role, auth_provider, created_at
		 FROM login_lookup($1, $2)`, orgSlug, email).Row()

	var (
		id       iamdomain.LoginIdentity
		role     string
		password *string
	)
	err := row.Scan(&id.OrgID, &id.UserID, &id.Email, &password, &role, &id.AuthProvider, &id.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, iamdomain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	id.PasswordHash = password
	id.Role = iamdomain.Role(role)
	return &id, nil
}

func scanUser(row orgScanner) (*iamdomain.User, error) {
	var (
		u        iamdomain.User
		role     string
		password *string
	)
	err := row.Scan(&u.ID, &u.OrgID, &u.Email, &role, &password, &u.AuthProvider, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, iamdomain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	u.Role = iamdomain.Role(role)
	if password != nil {
		u.PasswordHash = *password
	}
	return &u, nil
}

func mapDuplicate(err error, constraint string, target error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == constraint {
		return target
	}
	return err
}

func nilString(s string) any {
	if s == "" {
		return nil
	}
	return s
}

package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

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

// q resolves the query executor: the request transaction if present
// (set by the tenant-tx middleware), otherwise the pool.
func (r *PostgresIAMRepo) q(ctx context.Context) *gorm.DB {
	if tx, ok := db.TxFrom(ctx); ok {
		return tx
	}
	return r.pool.WithContext(ctx)
}

// tq is q but REQUIRES a tenant transaction — RLS-scoped tables must never be
// touched outside one, otherwise queries silently return zero rows.
func (r *PostgresIAMRepo) tq(ctx context.Context) (*gorm.DB, error) {
	tx, ok := db.TxFrom(ctx)
	if !ok {
		return nil, db.ErrNoTx
	}
	return tx, nil
}

func (r *PostgresIAMRepo) CreateOrg(ctx context.Context, org *iamdomain.Org) error {
	q, err := r.tq(ctx)
	if err != nil {
		return err
	}
	var weights []byte
	if org.ScoringWeights != nil {
		weights, _ = json.Marshal(org.ScoringWeights)
	}
	err = q.WithContext(ctx).Exec(
		`INSERT INTO orgs (id, name, slug, plan, scoring_weights, min_score_to_proceed, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		org.ID, org.Name, org.Slug, org.Plan, weights, org.MinScoreToProceed, org.CreatedAt).Error
	return mapDuplicate(err, "orgs_slug_key", iamdomain.ErrDuplicateSlug)
}

func (r *PostgresIAMRepo) GetOrg(ctx context.Context, id uuid.UUID) (*iamdomain.Org, error) {
	q, err := r.tq(ctx)
	if err != nil {
		return nil, err
	}
	row := q.Raw(
		`SELECT id, name, slug, plan, scoring_weights, min_score_to_proceed, created_at FROM orgs WHERE id = $1`, id).Row()
	return scanOrg(row)
}

func (r *PostgresIAMRepo) GetOrgBySlug(ctx context.Context, slug string) (*iamdomain.Org, error) {
	q, err := r.tq(ctx)
	if err != nil {
		return nil, err
	}
	row := q.Raw(
		`SELECT id, name, slug, plan, scoring_weights, min_score_to_proceed, created_at FROM orgs WHERE slug = $1`, slug).Row()
	return scanOrg(row)
}

type orgScanner interface {
	Scan(dest ...any) error
}

func scanOrg(row orgScanner) (*iamdomain.Org, error) {
	var (
		org      iamdomain.Org
		weights  []byte
		minScore *float64
	)
	err := row.Scan(&org.ID, &org.Name, &org.Slug, &org.Plan, &weights, &minScore, &org.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, iamdomain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	org.MinScoreToProceed = minScore
	if len(weights) > 0 {
		_ = json.Unmarshal(weights, &org.ScoringWeights)
	}
	return &org, nil
}

func (r *PostgresIAMRepo) CreateUser(ctx context.Context, user *iamdomain.User) error {
	q, err := r.tq(ctx)
	if err != nil {
		return err
	}
	err = q.WithContext(ctx).Exec(
		`INSERT INTO users (id, org_id, email, role, password_hash, auth_provider, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		user.ID, user.OrgID, user.Email, string(user.Role), nilString(user.PasswordHash), user.AuthProvider, user.CreatedAt).Error
	return mapDuplicate(err, "users_org_id_email_key", iamdomain.ErrDuplicateEmail)
}

func (r *PostgresIAMRepo) GetUserByID(ctx context.Context, id uuid.UUID) (*iamdomain.User, error) {
	q, err := r.tq(ctx)
	if err != nil {
		return nil, err
	}
	row := q.Raw(
		`SELECT id, org_id, email, role, password_hash, auth_provider, created_at FROM users WHERE id = $1`, id).Row()
	return scanUser(row)
}

func (r *PostgresIAMRepo) GetUserByEmail(ctx context.Context, orgID uuid.UUID, email string) (*iamdomain.User, error) {
	q, err := r.tq(ctx)
	if err != nil {
		return nil, err
	}
	row := q.Raw(
		`SELECT id, org_id, email, role, password_hash, auth_provider, created_at FROM users WHERE org_id = $1 AND email = $2`,
		orgID, email).Row()
	return scanUser(row)
}

func (r *PostgresIAMRepo) ListUsers(ctx context.Context, orgID uuid.UUID) ([]*iamdomain.User, error) {
	q, err := r.tq(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := q.Raw(
		`SELECT id, org_id, email, role, password_hash, auth_provider, created_at FROM users WHERE org_id = $1 ORDER BY created_at`,
		orgID).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

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

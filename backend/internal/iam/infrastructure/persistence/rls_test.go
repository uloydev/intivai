package persistence

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/google/uuid"
	iamdomain "github.com/intivai/backend/internal/iam/domain"
	"github.com/intivai/backend/pkg/db"
)

// Integration test — requires a live Postgres with migrations applied.
// Proves FORCE RLS: cross-tenant reads return nothing even for the table
// owner, and the pre-auth security-definer lookup still works.
// Run: TEST_DATABASE_URL=postgres://... go test ./internal/iam/infrastructure/persistence/
func TestTenantIsolation(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	pool, err := db.NewPool(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	repo := NewPostgresIAMRepo(pool)
	txm := NewPostgresTxManager(pool)

	slugA := "a" + uuid.NewString()[:8]
	slugB := "b" + uuid.NewString()[:8]
	orgA, userA := seedTenant(t, ctx, txm, repo, slugA)
	orgB, userB := seedTenant(t, ctx, txm, repo, slugB)

	// Same-tenant visibility: org A sees both its own users.
	var inA []*iamdomain.User
	if err := txm.RunInTx(ctx, &orgA.ID, func(tctx context.Context) error {
		inA, err = repo.ListUsers(tctx, orgA.ID)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if len(inA) != 2 {
		t.Fatalf("org A users = %d, want 2", len(inA))
	}

	// Cross-tenant: org A cannot read org B's user.
	var inB []*iamdomain.User
	if err := txm.RunInTx(ctx, &orgA.ID, func(tctx context.Context) error {
		_, err = repo.GetUserByID(tctx, userB.ID)
		if !errors.Is(err, iamdomain.ErrNotFound) {
			t.Fatalf("GetUserByID cross-tenant err = %v, want not found", err)
		}
		inB, err = repo.ListUsers(tctx, orgB.ID)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if len(inB) != 0 {
		t.Fatalf("org A sees %d users of org B, want 0", len(inB))
	}

	// Cross-tenant org lookup hidden by RLS.
	if err := txm.RunInTx(ctx, &orgA.ID, func(tctx context.Context) error {
		_, err = repo.GetOrg(tctx, orgB.ID)
		return err
	}); !errors.Is(err, iamdomain.ErrNotFound) {
		t.Fatalf("GetOrg cross-tenant err = %v, want not found", err)
	}

	// Pre-auth login lookup works without tenant context (security-definer).
	id, err := repo.FindLoginIdentity(ctx, orgA.Slug, userA.Email)
	if err != nil {
		t.Fatalf("login_lookup: %v", err)
	}
	if id.OrgID != orgA.ID || id.Role != iamdomain.RoleAdmin {
		t.Fatalf("login identity = %+v, want org %s admin", id, orgA.ID)
	}
}

func seedTenant(t *testing.T, ctx context.Context, txm *PostgresTxManager, repo *PostgresIAMRepo, slug string) (*iamdomain.Org, *iamdomain.User) {
	t.Helper()
	org, err := iamdomain.NewOrg("Isolation Test "+slug, slug)
	if err != nil {
		t.Fatal(err)
	}
	admin, err := iamdomain.NewUser(org.ID, "admin@"+slug+".io", iamdomain.RoleAdmin, "hash-"+uuid.NewString(), "password")
	if err != nil {
		t.Fatal(err)
	}
	recruiter, err := iamdomain.NewUser(org.ID, "recruiter@"+slug+".io", iamdomain.RoleRecruiter, "hash-"+uuid.NewString(), "password")
	if err != nil {
		t.Fatal(err)
	}
	if err := txm.RunInTx(ctx, &org.ID, func(tctx context.Context) error {
		if err := repo.CreateOrg(tctx, org); err != nil {
			return err
		}
		if err := repo.CreateUser(tctx, admin); err != nil {
			return err
		}
		return repo.CreateUser(tctx, recruiter)
	}); err != nil {
		t.Fatal(err)
	}
	return org, admin
}

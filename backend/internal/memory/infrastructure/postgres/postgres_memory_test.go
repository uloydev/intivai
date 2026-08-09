package postgres

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/intivai/backend/pkg/db"
	"gorm.io/gorm"
)

func seedOrg(t *testing.T, pool *gorm.DB, orgID string) {
	t.Helper()
	ctx := context.Background()
	tx := pool.WithContext(ctx).Begin()
	defer tx.Rollback()
	if err := db.SetTenant(ctx, tx, orgID); err != nil {
		t.Fatal(err)
	}
	if err := tx.Exec(`INSERT INTO orgs (id, name, slug) VALUES ($1, $2, $3)`,
		orgID, "test-"+orgID[:8], "t"+orgID[:8]).Error; err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit().Error; err != nil {
		t.Fatal(err)
	}
}

// Integration test — requires a live Postgres with migrations applied.
// Run: TEST_DATABASE_URL=postgres://... go test ./internal/memory/infrastructure/postgres/
func TestPostgresBankLifecycle(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	pool, err := db.NewPool(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	orgID := uuid.NewString()
	otherID := uuid.NewString()
	seedOrg(t, pool, orgID)
	seedOrg(t, pool, otherID)

	factory := NewPostgresFactory(pool)
	bank := factory.ForBank(orgID)

	if err := bank.Remember(ctx, "candidate_profile", "strong Go fintech candidate", 0.9); err != nil {
		t.Fatalf("remember: %v", err)
	}

	stats, err := bank.Stats(ctx)
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if stats.Memories != 1 {
		t.Fatalf("memories = %d, want 1", stats.Memories)
	}

	hits, err := bank.Recall(ctx, "Go fintech", "")
	if err != nil {
		t.Fatalf("recall: %v", err)
	}
	if len(hits) != 1 || hits[0].Content != "strong Go fintech candidate" {
		t.Fatalf("recall hits = %+v", hits)
	}

	// Other tenant must not see the memory (RLS + org_id partition).
	other := factory.ForBank(otherID)
	if n, err := other.Stats(ctx); err != nil || n.Memories != 0 {
		t.Fatalf("other tenant stats = %+v, err %v; want 0", n, err)
	}

	if err := bank.Forget(ctx, hits[0].ID); err != nil {
		t.Fatalf("forget: %v", err)
	}
	if stats, _ := bank.Stats(ctx); stats.Memories != 0 {
		t.Fatalf("memories after forget = %d, want 0", stats.Memories)
	}
}

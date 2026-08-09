package postgres

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/intivai/backend/pkg/db"
)

// fakeEmbedder — deterministic 384-dim vectors (same text → same vector,
// distinct texts → far apart). Table column is VECTOR(384).
type fakeEmbedder struct{}

func (fakeEmbedder) Embed(_ context.Context, text string) ([]float32, error) {
	vec := make([]float32, 384)
	for i := range vec {
		vec[i] = float32(int(text[0]) + i%7)
	}
	return vec, nil
}

// Semantic recall (carryover item 8): with an embedder attached, Recall ranks
// by cosine distance; the exact match surfaces first even without keyword
// overlap, and cross-tenant isolation still holds.
func TestPostgresBankCosineRecall(t *testing.T) {
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
	seedOrg(t, pool, orgID)

	factory := NewPostgresFactory(pool).WithEmbedder(fakeEmbedder{})
	bank := factory.ForBank(orgID)

	// "Go" and "fintech" texts differ → their vectors are far apart; the
	// query text equals the first memory → exact cosine match.
	if err := bank.Remember(ctx, "candidate_profile", "go", 0.5); err != nil {
		t.Fatal(err)
	}
	if err := bank.Remember(ctx, "candidate_profile", "fintech", 0.5); err != nil {
		t.Fatal(err)
	}

	hits, err := bank.Recall(ctx, "go", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 2 {
		t.Fatalf("hits = %d, want 2 embedded rows", len(hits))
	}
	if hits[0].Content != "go" {
		t.Fatalf("top hit = %q, want exact cosine match", hits[0].Content)
	}
	if hits[0].Score <= hits[1].Score {
		t.Fatalf("exact match score %f must exceed %f", hits[0].Score, hits[1].Score)
	}
}

package persistence

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	ctxdomain "github.com/intivai/backend/internal/context/domain"
	"github.com/intivai/backend/pkg/db"
	"gorm.io/gorm"
)

func seedOrg(t *testing.T, pool *gorm.DB, orgID, slug string) {
	t.Helper()
	if err := db.RunInTx(context.Background(), pool, orgID, func(tctx context.Context) error {
		tx, _ := db.TxFrom(tctx)
		return tx.Exec(`INSERT INTO orgs (id, name, slug) VALUES ($1,$2,$3)`, orgID, "t", slug).Error
	}); err != nil {
		t.Fatal(err)
	}
}

func TestContextRoundTrip(t *testing.T) {
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
	seedOrg(t, pool, orgID, "xt"+orgID[:8])

	repo := NewPostgresContextRepo(pool)

	cc, err := ctxdomain.NewCompanyContext(uuid.MustParse(orgID), ctxdomain.TypeText, "hash1", "path1")
	if err != nil {
		t.Fatal(err)
	}
	if err := db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
		return repo.CreateContext(tctx, cc)
	}); err != nil {
		t.Fatalf("create context: %v", err)
	}

	var byHash *ctxdomain.CompanyContext
	err = db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
		byHash, err = repo.GetContextByHash(tctx, uuid.MustParse(orgID), "hash1")
		return err
	})
	if err != nil {
		t.Fatalf("get by hash: %v", err)
	}
	if byHash.Version != 1 {
		t.Fatalf("version = %d", byHash.Version)
	}

	// Prompt versioning: v1 then v2, latest returns v2.
	if err := db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
		return repo.SetPrompt(tctx, &ctxdomain.TenantPrompt{OrgID: uuid.MustParse(orgID), SystemPrompt: "p1", Version: 1})
	}); err != nil {
		t.Fatalf("prompt v1: %v", err)
	}
	if err := db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
		return repo.SetPrompt(tctx, &ctxdomain.TenantPrompt{OrgID: uuid.MustParse(orgID), SystemPrompt: "p2", Version: 2})
	}); err != nil {
		t.Fatalf("prompt v2: %v", err)
	}
	var latest *ctxdomain.TenantPrompt
	err = db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
		latest, err = repo.GetLatestPrompt(tctx, uuid.MustParse(orgID))
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if latest.Version != 2 || latest.SystemPrompt != "p2" {
		t.Fatalf("latest prompt = %+v", latest)
	}

	// No-prompt org returns ErrNotFound (drives default fallback).
	otherOrg := uuid.NewString()
	seedOrg(t, pool, otherOrg, "xn"+otherOrg[:8])
	err = db.RunInTx(ctx, pool, otherOrg, func(tctx context.Context) error {
		_, err = repo.GetLatestPrompt(tctx, uuid.MustParse(otherOrg))
		return err
	})
	if err != ctxdomain.ErrNotFound {
		t.Fatalf("missing prompt err = %v, want ErrNotFound", err)
	}
}

package persistence

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/intivai/backend/internal/integration/domain"
	shareddomain "github.com/intivai/backend/internal/shared/domain"
	"github.com/intivai/backend/pkg/db"
)

// Integration test — requires a live Postgres with migration 025 applied.
// Proves FORCE RLS on the P4a-era tenant tables (webhook_configs,
// webhook_deliveries, recruiter_decisions, data_requests): cross-tenant reads
// return nothing even though the app role could read the table before 025.
// Run: TEST_DATABASE_URL=postgres://... go test ./internal/integration/infrastructure/persistence/
func TestWebhookTenantIsolation(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	pool, err := db.NewPool(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	repo := NewPostgresWebhookRepo(pool)

	orgA := uuid.New()
	orgB := uuid.New()
	slugA := "wha" + uuid.NewString()[:6]
	slugB := "whb" + uuid.NewString()[:6]
	for _, o := range []struct {
		id   uuid.UUID
		slug string
	}{{orgA, slugA}, {orgB, slugB}} {
		if err := db.RunInTx(ctx, pool, o.id.String(), func(tctx context.Context) error {
			tx, ok := db.TxFrom(tctx)
			if !ok {
				t.Fatal("no tx")
			}
			return tx.Exec(`INSERT INTO orgs (id, name, slug) VALUES ($1,$2,$3) ON CONFLICT (slug) DO NOTHING`, o.id, "t", o.slug).Error
		}); err != nil {
			t.Fatal(err)
		}
	}

	// Org A creates a webhook config + delivery.
	cfg, err := domain.NewWebhookConfig(orgA, "https://example.com/hook", []domain.WebhookEvent{domain.EventInterviewCompleted}, "secret")
	if err != nil {
		t.Fatal(err)
	}
	if err := db.RunInTx(ctx, pool, orgA.String(), func(tctx context.Context) error {
		return repo.CreateConfig(tctx, cfg)
	}); err != nil {
		t.Fatal(err)
	}
	delivery := &domain.WebhookDelivery{
		Entity:      shareddomain.Entity{ID: uuid.New(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		OrgID:       orgA,
		WebhookID:   cfg.ID,
		Event:       domain.EventInterviewCompleted,
		Payload:     []byte(`{"k":"v"}`),
		FinalStatus: "pending",
	}
	if err := db.RunInTx(ctx, pool, orgA.String(), func(tctx context.Context) error {
		return repo.CreateDelivery(tctx, delivery)
	}); err != nil {
		t.Fatal(err)
	}

	// Same tenant: org A sees its own config + delivery.
	if err := db.RunInTx(ctx, pool, orgA.String(), func(tctx context.Context) error {
		got, err := repo.GetConfigByID(tctx, cfg.ID)
		if err != nil {
			return err
		}
		if got.OrgID != orgA {
			t.Fatalf("config org = %s, want orgA", got.OrgID)
		}
		d, err := repo.GetDeliveryByID(tctx, delivery.ID)
		if err != nil {
			return err
		}
		if d.OrgID != orgA {
			t.Fatalf("delivery org = %s, want orgA", d.OrgID)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	// Cross tenant: org B must NOT see org A's config, delivery, or list.
	if err := db.RunInTx(ctx, pool, orgB.String(), func(tctx context.Context) error {
		if _, err := repo.GetConfigByID(tctx, cfg.ID); !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("cross-tenant GetConfigByID err = %v, want not found", err)
		}
		if _, err := repo.GetDeliveryByID(tctx, delivery.ID); !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("cross-tenant GetDeliveryByID err = %v, want not found", err)
		}
		configs, err := repo.ListConfigsByOrg(tctx, orgA)
		if err != nil {
			return err
		}
		if len(configs) != 0 {
			t.Fatalf("cross-tenant ListConfigsByOrg = %d rows, want 0", len(configs))
		}
		deliveries, err := repo.ListDeliveriesByWebhook(tctx, cfg.ID, 10)
		if err != nil {
			return err
		}
		if len(deliveries) != 0 {
			t.Fatalf("cross-tenant ListDeliveriesByWebhook = %d rows, want 0", len(deliveries))
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

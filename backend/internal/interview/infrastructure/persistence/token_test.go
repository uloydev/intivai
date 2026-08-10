package persistence

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	ivdomain "github.com/intivai/backend/internal/interview/domain"
	"github.com/intivai/backend/pkg/db"
)

func randToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// Token lifecycle: create → validate (definer, no tenant ctx) → mark used →
// validate again shows 'used'. Also: unknown token → not_found.
func TestTokenLifecycle(t *testing.T) {
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
	appID := uuid.New()
	jobID := uuid.New()
	candID := uuid.New()
	seedOrg(t, pool, orgID, "tk"+orgID[:8])

	err = db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
		tx, _ := db.TxFrom(tctx)
		if err := tx.Exec(`INSERT INTO jobs (id, org_id, title, description, status, created_at) VALUES ($1,$2,$3,$4,'active',NOW())`, jobID, orgID, "J", "d").Error; err != nil {
			return err
		}
		if err := tx.Exec(`INSERT INTO candidates (id, org_id, name, status, created_at) VALUES ($1,$2,$3,'extracted',NOW())`, candID, orgID, "Jane").Error; err != nil {
			return err
		}
		return tx.Exec(`INSERT INTO applications (id, org_id, candidate_id, job_id, status, created_at) VALUES ($1,$2,$3,$4,'passed',NOW())`, appID, orgID, candID, jobID).Error
	})
	if err != nil {
		t.Fatal(err)
	}

	tokenRepo := NewPostgresTokenRepo(pool)
	invite := &ivdomain.InvitationToken{
		ID: uuid.New(), OrgID: uuid.MustParse(orgID), InterviewID: uuid.Nil,
		Token: randToken(), ExpiresAt: time.Now().UTC().Add(7 * 24 * time.Hour),
	}
	// Create needs an interview row.
	ivRepo := NewPostgresInterviewRepo(pool)
	iv, err := ivdomain.NewInterview(uuid.MustParse(orgID), appID, []ivdomain.Question{{Idx: 1, Content: "Q", Category: "technical"}}, time.Now().UTC().Add(time.Hour), ivdomain.SystemClock())
	if err != nil {
		t.Fatal(err)
	}
	if err := db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
		return ivRepo.Create(tctx, iv)
	}); err != nil {
		t.Fatal(err)
	}
	invite.InterviewID = iv.ID
	if err := db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
		return tokenRepo.Create(tctx, invite)
	}); err != nil {
		t.Fatalf("create token: %v", err)
	}

	// Pre-auth validation — plain ctx, no tenant tx.
	validated, status := tokenRepo.Validate(ctx, invite.Token)
	if status != ivdomain.TokenValid || validated.InterviewID != iv.ID {
		t.Fatalf("validate = %s (%+v)", status, validated)
	}

	if err := db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
		return tokenRepo.MarkUsed(tctx, invite.Token)
	}); err != nil {
		t.Fatalf("mark used: %v", err)
	}
	if _, status := tokenRepo.Validate(ctx, invite.Token); status != ivdomain.TokenUsed {
		t.Fatalf("after use = %s, want used", status)
	}

	if _, status := tokenRepo.Validate(ctx, randToken()); status != ivdomain.TokenNotFound {
		t.Fatalf("unknown token = %s, want not_found", status)
	}
}

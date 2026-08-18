package persistence

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	ivdomain "github.com/intivai/backend/internal/interview/domain"
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

// Round-trip: interview create (NULL optional fields) → get → update with
// transcript/status. Guards NULL scans + jsonb transcript.
func TestInterviewRoundTrip(t *testing.T) {
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
	seedOrg(t, pool, orgID, "iv"+orgID[:8])

	// application FK: create job + candidate + application chain
	jobID := uuid.New()
	candID := uuid.New()
	appID := uuid.New()
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

	repo := NewPostgresInterviewRepo(pool)
	iv, err := ivdomain.NewInterview(uuid.MustParse(orgID), appID, []ivdomain.Question{
		{Idx: 1, Content: "Q1", Category: "technical", Skill: "Go"},
	}, time.Now().UTC().Add(time.Hour), ivdomain.SystemClock())
	if err != nil {
		t.Fatal(err)
	}
	if err := db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
		return repo.Create(tctx, iv)
	}); err != nil {
		t.Fatalf("create: %v", err)
	}

	var got *ivdomain.Interview
	err = db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
		got, err = repo.GetByID(tctx, iv.ID)
		return err
	})
	if err != nil {
		t.Fatalf("get (NULL optional scans): %v", err)
	}
	if got.Status != ivdomain.StatusPending || len(got.Questions) != 1 {
		t.Fatalf("round-trip mismatch: %+v", got)
	}

	_ = got.Start()
	_ = got.Answer("my answer")
	if err := db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
		return repo.Update(tctx, got)
	}); err != nil {
		t.Fatalf("update: %v", err)
	}
	err = db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
		got, err = repo.GetByID(tctx, iv.ID)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != ivdomain.StatusInProgress || got.LastQuestionIdx != 1 || len(got.Answers) != 1 {
		t.Fatalf("updated mismatch: status=%s idx=%d answers=%d", got.Status, got.LastQuestionIdx, len(got.Answers))
	}
}

func TestQuestionBankRoundTrip(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	pool, err := db.NewPool(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	orgID := uuid.New()
	seedOrg(t, pool, orgID.String(), "qb"+orgID.String()[:8])

	bank := NewPostgresQuestionBank(pool)
	q := ivdomain.Question{
		Content:  "How do channels coordinate goroutines?",
		Category: "technical",
		Skill:    "Go",
	}

	if err := db.RunInTx(ctx, pool, orgID.String(), func(tctx context.Context) error {
		return bank.Create(tctx, orgID, q)
	}); err != nil {
		t.Fatalf("bank create: %v", err)
	}

	var list []ivdomain.Question
	if err := db.RunInTx(ctx, pool, orgID.String(), func(tctx context.Context) error {
		var err error
		list, err = bank.ListByOrg(tctx, orgID)
		return err
	}); err != nil {
		t.Fatalf("bank list: %v", err)
	}

	if len(list) == 0 || list[0].Content != q.Content || list[0].Skill != q.Skill {
		t.Fatalf("unexpected question bank results: %+v", list)
	}
}

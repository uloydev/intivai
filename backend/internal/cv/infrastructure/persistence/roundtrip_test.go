package persistence

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	cvdomain "github.com/intivai/backend/internal/cv/domain"
	shareddomain "github.com/intivai/backend/internal/shared/domain"
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

// Round-trip: candidate with all-NULL optional columns, then structured
// update, list, delete. Guards NULL scans (cv_ocr_method, raw text, error).
func TestCandidateRoundTrip(t *testing.T) {
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
	seedOrg(t, pool, orgID, "ct"+orgID[:8])

	repo := NewPostgresCandidateRepo(pool)
	c := &cvdomain.Candidate{
		Entity: shareddomain.Entity{ID: uuid.New(), CreatedAt: time.Now().UTC()},
		OrgID:  uuid.MustParse(orgID), Name: "Jane", Email: "j@x.io", Status: cvdomain.StatusParsing,
	}
	if err := db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
		return repo.Create(tctx, c)
	}); err != nil {
		t.Fatalf("create: %v", err)
	}

	var got *cvdomain.Candidate
	err = db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
		got, err = repo.GetByID(tctx, c.ID)
		return err
	})
	if err != nil {
		t.Fatalf("get (all NULL optionals): %v", err)
	}
	if got.CVRawText != "" || got.CVOCRMethod != "" || got.ErrorMessage != "" {
		t.Fatalf("NULL columns leaked: %+v", got)
	}

	got.CVRawText = "raw cv text"
	got.CVStructured = []byte(`{"skills":["Go"]}`)
	got.CVOCRMethod = "pdfcpu"
	got.Status = cvdomain.StatusExtracted
	if err := db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
		return repo.Update(tctx, got)
	}); err != nil {
		t.Fatalf("update: %v", err)
	}

	var list []*cvdomain.Candidate
	err = db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
		list, err = repo.List(tctx, uuid.MustParse(orgID))
		return err
	})
	if err != nil || len(list) != 1 {
		t.Fatalf("list: %d rows, err %v", len(list), err)
	}
	var structured struct {
		Skills []string `json:"skills"`
	}
	if err := json.Unmarshal(list[0].CVStructured, &structured); err != nil {
		t.Fatalf("structured not json: %s", list[0].CVStructured)
	}
	if len(structured.Skills) != 1 || structured.Skills[0] != "Go" {
		t.Fatalf("structured round-trip: %s", list[0].CVStructured)
	}

	if err := db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
		return repo.Delete(tctx, c.ID)
	}); err != nil {
		t.Fatalf("delete: %v", err)
	}
}

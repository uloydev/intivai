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

// Public review flow: token lookup must work with NO tenant transaction
// (SECURITY DEFINER function) and decode every column incl. cv_format.
// Regression: candidate_by_review_token did not expose cv_format while the
// repo selected COALESCE(cv_format,'pdf') → plan-time 42703 → API 500.
func TestGetByReviewTokenPublicLookup(t *testing.T) {
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
	seedOrg(t, pool, orgID, "rt"+orgID[:8])

	repo := NewPostgresCandidateRepo(pool)
	token := uuid.NewString()
	c := &cvdomain.Candidate{
		Entity: shareddomain.Entity{ID: uuid.New(), CreatedAt: time.Now().UTC()},
		OrgID:  uuid.MustParse(orgID), Name: "Review", Email: "r@x.io",
		Status:    cvdomain.StatusPendingReview,
		CVRawText: "raw cv", CVStructured: json.RawMessage(`{"skills":["Go"]}`),
		ReviewToken: &token,
	}
	if err := db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
		if err := repo.Create(tctx, c); err != nil {
			return err
		}
		// Create persists identity/status only; payload lands via Update
		// (mirrors the worker pipeline).
		return repo.Update(tctx, c)
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	got, err := repo.GetByReviewToken(ctx, token)
	if err != nil {
		t.Fatalf("get by review token (public, no tenant tx): %v", err)
	}
	if got.ID != c.ID || got.OrgID != c.OrgID {
		t.Fatalf("identity mismatch: got %+v", got)
	}
	if got.Status != cvdomain.StatusPendingReview {
		t.Fatalf("status: got %q want pending_review", got.Status)
	}
	if got.ReviewToken == nil || *got.ReviewToken != token {
		t.Fatalf("review token mismatch: %v", got.ReviewToken)
	}
	if got.CVRawText != "raw cv" {
		t.Fatalf("payload round-trip raw: %q", got.CVRawText)
	}
	var skills struct {
		Skills []string `json:"skills"`
	}
	if err := json.Unmarshal(got.CVStructured, &skills); err != nil || len(skills.Skills) != 1 || skills.Skills[0] != "Go" {
		t.Fatalf("payload round-trip structured: %s (%v)", got.CVStructured, err)
	}
	if got.CVFormat != "pdf" {
		t.Fatalf("cv_format default: got %q want pdf", got.CVFormat)
	}

	if _, err := repo.GetByReviewToken(ctx, uuid.NewString()); err != cvdomain.ErrNotFound {
		t.Fatalf("unknown token: want ErrNotFound, got %v", err)
	}
}

// D1: confirm persists candidate-edited name/email alongside the structured
// profile atomically (single UPDATE in the SECURITY DEFINER function), clears
// the token, and is a no-op (uuid.Nil) on replay. Runs OUTSIDE any tenant tx
// — the confirm endpoint is public.
func TestConfirmReviewUpdatesNameEmail(t *testing.T) {
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
	seedOrg(t, pool, orgID, "crm"+orgID[:8])

	repo := NewPostgresCandidateRepo(pool)
	token := uuid.NewString()
	c := &cvdomain.Candidate{
		Entity: shareddomain.Entity{ID: uuid.New(), CreatedAt: time.Now().UTC()},
		OrgID:  uuid.MustParse(orgID), Name: "Review", Email: "r@x.io",
		Status:    cvdomain.StatusPendingReview,
		CVRawText: "raw cv", CVStructured: json.RawMessage(`{"skills":["Go"]}`),
		ReviewToken: &token,
	}
	if err := db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
		if err := repo.Create(tctx, c); err != nil {
			return err
		}
		return repo.Update(tctx, c)
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	structured := []byte(`{"skills":["Go","SQL"],"experience_years":5,"education":"MSc","certifications":["AWS"],"summary":"backend"}`)
	orgIDGot, candIDGot, err := repo.ConfirmReview(ctx, token, structured, "New Name", "new@x.io")
	if err != nil {
		t.Fatalf("confirm: %v", err)
	}
	if orgIDGot != c.OrgID || candIDGot != c.ID {
		t.Fatalf("confirm returned %s/%s, want %s/%s", orgIDGot, candIDGot, c.OrgID, c.ID)
	}

	if _, err := repo.GetByReviewToken(ctx, token); err != cvdomain.ErrNotFound {
		t.Fatalf("token not cleared: want ErrNotFound, got %v", err)
	}

	var got *cvdomain.Candidate
	err = db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
		got, err = repo.GetByID(tctx, c.ID)
		return err
	})
	if err != nil {
		t.Fatalf("get after confirm: %v", err)
	}
	if got.Name != "New Name" || got.Email != "new@x.io" {
		t.Fatalf("identity not updated: %+v", got)
	}
	if got.Status != cvdomain.StatusExtracted {
		t.Fatalf("status = %q, want extracted", got.Status)
	}
	var rd struct {
		Skills []string `json:"skills"`
	}
	if err := json.Unmarshal(got.CVStructured, &rd); err != nil || len(rd.Skills) != 2 || rd.Skills[0] != "Go" {
		t.Fatalf("structured not persisted: %s (%v)", got.CVStructured, err)
	}

	org2, cand2, err := repo.ConfirmReview(ctx, token, structured, "Nope", "n@x.io")
	if err != nil {
		t.Fatalf("replay confirm must not error: %v", err)
	}
	if org2 != uuid.Nil || cand2 != uuid.Nil {
		t.Fatalf("replay confirm returned %s/%s, want nil/nil", org2, cand2)
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

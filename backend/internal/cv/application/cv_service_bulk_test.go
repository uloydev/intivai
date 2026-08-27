package application

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/intivai/backend/internal/cv/domain"
)

// Dedicated bulk fixtures — failures are injected per-file so a single batch
// can exercise every skip path deterministically.
type bulkStoreFixture struct {
	failCT    string
	uploadErr error
}

func (s *bulkStoreFixture) Upload(ctx context.Context, path string, r io.Reader, size int64, ct string) error {
	if s.failCT != "" && ct == s.failCT {
		return s.uploadErr
	}
	return nil
}
func (s *bulkStoreFixture) Download(ctx context.Context, path string) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("")), nil
}
func (s *bulkStoreFixture) Delete(ctx context.Context, path string) error { return nil }

type bulkRepoFixture struct {
	failName string
	created  []*domain.Candidate
	deleted  []uuid.UUID
}

func (r *bulkRepoFixture) Create(ctx context.Context, c *domain.Candidate) error {
	if c.Name == r.failName {
		return errors.New("duplicate candidate")
	}
	r.created = append(r.created, c)
	return nil
}
func (r *bulkRepoFixture) Delete(ctx context.Context, id uuid.UUID) error {
	r.deleted = append(r.deleted, id)
	return nil
}

func (r *bulkRepoFixture) ListByIDs(ctx context.Context, orgID uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]*domain.Candidate, error) {
	return map[uuid.UUID]*domain.Candidate{}, nil
}
func (r *bulkRepoFixture) ConfirmReview(ctx context.Context, token string, structured []byte, name, email string) (uuid.UUID, uuid.UUID, error) {
	return uuid.Nil, uuid.Nil, domain.ErrNotFound
}
func (r *bulkRepoFixture) GetByID(ctx context.Context, id uuid.UUID) (*domain.Candidate, error) {
	return nil, domain.ErrNotFound
}
func (r *bulkRepoFixture) GetByReviewToken(ctx context.Context, token string) (*domain.Candidate, error) {
	return nil, domain.ErrNotFound
}
func (r *bulkRepoFixture) List(ctx context.Context, orgID uuid.UUID) ([]*domain.Candidate, error) {
	return r.created, nil
}
func (r *bulkRepoFixture) Update(ctx context.Context, c *domain.Candidate) error { return nil }

const bulkStorageTriggerCT = "test/fail-storage"

// Every submitted file must come back as an explicit outcome row — silent
// drops hide ingestion problems from recruiters (finding D27).
func TestBulkUploadPerFileOutcomes(t *testing.T) {
	repo := &bulkRepoFixture{failName: "Persist.pdf"}
	svc := NewCVService(repo, nil, &bulkStoreFixture{failCT: bulkStorageTriggerCT, uploadErr: errors.New("minio down")}, &stubEnqueuer{}, nil)

	files := []BulkUploadFile{
		{Name: "Good.pdf", Data: []byte("pdf"), ContentType: "application/pdf"},
		{Name: "   ", Data: []byte("pdf"), ContentType: "application/pdf"},            // blank name → invalid
		{Name: "Storage.pdf", Data: []byte("pdf"), ContentType: bulkStorageTriggerCT}, // store fails
		{Name: "Persist.pdf", Data: []byte("pdf"), ContentType: "application/pdf"},    // repo fails
	}

	res, err := svc.BulkUpload(context.Background(), actor(), files)
	if err != nil {
		t.Fatalf("partial failure must still return the report: %v", err)
	}
	if res.Uploaded != 1 || res.Failed != 3 {
		t.Fatalf("counts uploaded=%d failed=%d, want 1/3", res.Uploaded, res.Failed)
	}
	if len(res.Outcomes) != 4 {
		t.Fatalf("outcomes = %d, want one per submitted file", len(res.Outcomes))
	}
	want := map[string]struct {
		ok     bool
		reason string
	}{
		"Good.pdf":    {true, ""},
		"   ":         {false, "invalid_candidate_name"},
		"Storage.pdf": {false, "storage_failed"},
		"Persist.pdf": {false, "persist_failed"},
	}
	for _, o := range res.Outcomes {
		w := want[o.Filename]
		if o.OK != w.ok || o.Reason != w.reason {
			t.Fatalf("outcome %+v, want ok=%v reason=%q", o, w.ok, w.reason)
		}
	}
}

// Enqueue failure must surface too — compensation happens AND the recruiter
// learns the file was not ingested.
func TestBulkUploadEnqueueFailureReported(t *testing.T) {
	repo := &bulkRepoFixture{}
	svc := NewCVService(repo, nil, &bulkStoreFixture{}, &stubEnqueuer{fail: true}, nil)

	res, err := svc.BulkUpload(context.Background(), actor(), []BulkUploadFile{
		{Name: "Queued.pdf", Data: []byte("pdf"), ContentType: "application/pdf"},
	})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if res.Failed != 1 || res.Outcomes[0].Reason != "enqueue_failed" || res.Outcomes[0].OK {
		t.Fatalf("outcome missing enqueue failure: %+v", res.Outcomes)
	}
	if len(repo.deleted) != 1 {
		t.Fatal("compensation (candidate delete) lost")
	}
}

// All-failed stays a 201-shaped success carrying the batch id and the full
// outcome list — the recruiter sees exactly why nothing landed.
func TestBulkUploadAllFailedStillReturnsBatch(t *testing.T) {
	svc := NewCVService(&bulkRepoFixture{}, nil, &bulkStoreFixture{}, &stubEnqueuer{}, nil)

	res, err := svc.BulkUpload(context.Background(), actor(), []BulkUploadFile{
		{Name: "", Data: []byte("pdf"), ContentType: "application/pdf"},
		{Name: "  ", Data: []byte("pdf"), ContentType: "application/pdf"},
	})
	if err != nil {
		t.Fatalf("all-failed must not error: %v", err)
	}
	if res.BatchID == uuid.Nil {
		t.Fatal("batch id missing")
	}
	if res.Uploaded != 0 || res.Failed != 2 || len(res.Outcomes) != 2 {
		t.Fatalf("counts wrong: %+v", res)
	}
}

// Response contract (openapi.yaml /cvs/bulk): batch_id, uploaded, failed,
// outcomes[{filename, ok, reason}].
func TestBulkUploadResultJSONShape(t *testing.T) {
	res := &BulkUploadResult{
		BatchID:  uuid.New(),
		Uploaded: 1,
		Failed:   1,
		Outcomes: []BulkUploadOutcome{
			{Filename: "a.pdf", OK: true},
			{Filename: "b.pdf", OK: false, Reason: "storage_failed"},
		},
	}
	raw, err := json.Marshal(res)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"batch_id", "uploaded", "failed", "outcomes"} {
		if _, ok := m[key]; !ok {
			t.Fatalf("response json missing %q: %s", key, raw)
		}
	}
	outcomes, _ := m["outcomes"].([]any)
	first, _ := outcomes[0].(map[string]any)
	for _, key := range []string{"filename", "ok"} {
		if _, ok := first[key]; !ok {
			t.Fatalf("outcome json missing %q: %s", key, raw)
		}
	}
}

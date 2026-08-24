package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"

	cvdomain "github.com/intivai/backend/internal/cv/domain"
	"github.com/intivai/backend/pkg/db"
	"github.com/rs/zerolog"
	"gorm.io/gorm"
)

func txCtx() context.Context {
	return db.WithTx(context.Background(), &gorm.DB{})
}

// recordingCandidateRepo captures Update calls so tests can assert exactly
// what the worker persisted (status + error_message).
type recordingCandidateRepo struct {
	*stubRepo
	updates []cvdomain.Candidate
}

func (r *recordingCandidateRepo) Update(ctx context.Context, c *cvdomain.Candidate) error {
	r.updates = append(r.updates, *c)
	return r.stubRepo.Update(ctx, c)
}

// stubFileStorage replaces the concrete MinIO handle in ParseWorker.
type stubFileStorage struct {
	data  []byte
	dlErr error
}

func (s *stubFileStorage) Upload(ctx context.Context, path string, r io.Reader, size int64, ct string) error {
	return nil
}
func (s *stubFileStorage) Download(ctx context.Context, path string) (io.ReadCloser, error) {
	if s.dlErr != nil {
		return nil, s.dlErr
	}
	return io.NopCloser(strings.NewReader(string(s.data))), nil
}
func (s *stubFileStorage) Delete(ctx context.Context, path string) error { return nil }
func (s *stubFileStorage) Exists(ctx context.Context, path string) (bool, error) {
	return true, nil
}

func newTestParseWorker(repo *recordingCandidateRepo, store *stubFileStorage) *ParseWorker {
	return &ParseWorker{repo: repo, store: store, queue: parseQueueStub{}, log: zerolog.Nop()}
}

func parseTask(orgID uuid.UUID, candID uuid.UUID) *asynq.Task {
	payload, _ := json.Marshal(ParseCVPayload{OrgID: orgID.String(), CandidateID: candID.String()})
	return asynq.NewTask(TaskParseCV, payload)
}

func seedParsingCandidate(repo *recordingCandidateRepo, orgID uuid.UUID) cvdomain.Candidate {
	cand := cvdomain.Candidate{
		Entity: entity(uuid.New()), OrgID: orgID,
		Name: "Jane", Status: cvdomain.StatusParsing, CVPath: "cvs/x/y.pdf",
	}
	repo.created = append(repo.created, &cand)
	return cand
}

// A transient object-store failure must NOT burn the candidate as terminal
// failed_ocr — MinIO blips recover, so the task stays retryable and the
// candidate stays parsing (finding D26 / plan 5.7).
func TestParseWorkerDownloadFailureIsTransient(t *testing.T) {
	repo := &recordingCandidateRepo{stubRepo: &stubRepo{}}
	orgID := uuid.New()
	cand := seedParsingCandidate(repo, orgID)

	store := &stubFileStorage{dlErr: errors.New("minio: server timeout")}
	err := newTestParseWorker(repo, store).handle(txCtx(), parseTask(orgID, cand.ID))

	if err == nil || errors.Is(err, asynq.SkipRetry) {
		t.Fatalf("download failure returned %v, want retryable error", err)
	}
	for i, u := range repo.updates {
		if u.Status == cvdomain.StatusFailedOCR {
			t.Fatalf("update %d marked candidate failed_ocr on transient store fault", i)
		}
	}
}

// True unreadable content is terminal failed_ocr — but the persisted
// error_message must be a safe category, not raw parser text (finding D26).
func TestParseWorkerUnreadableContentMarksSafeMessage(t *testing.T) {
	repo := &recordingCandidateRepo{stubRepo: &stubRepo{}}
	orgID := uuid.New()
	cand := seedParsingCandidate(repo, orgID)

	store := &stubFileStorage{data: []byte("this is not a pdf at all")}
	err := newTestParseWorker(repo, store).handle(txCtx(), parseTask(orgID, cand.ID))

	if !errors.Is(err, asynq.SkipRetry) {
		t.Fatalf("unreadable content is permanent, got %v", err)
	}
	if len(repo.updates) == 0 {
		t.Fatal("no update persisted")
	}
	last := repo.updates[len(repo.updates)-1]
	if last.Status != cvdomain.StatusFailedOCR {
		t.Fatalf("status = %q, want failed_ocr", last.Status)
	}
	want := "cv_unreadable — file could not be parsed as PDF or DOCX"
	if last.ErrorMessage != want {
		t.Fatalf("error_message = %q, want safe category %q", last.ErrorMessage, want)
	}
}

// The extract worker's fail path must persist a category, never the raw
// cause (which carries provider endpoints/timeouts) into error_message.
func TestExtractFailPersistsSafeCategoryOnly(t *testing.T) {
	repo := &recordingCandidateRepo{stubRepo: &stubRepo{}}
	orgID := uuid.New()
	cand := seedParsingCandidate(repo, orgID)
	w := &ExtractWorker{candRepo: repo, log: zerolog.Nop()}

	rawCause := fmt.Errorf("%w: POST https://api.provider.test/v1: dial tcp 10.0.0.1:443 i/o timeout", ErrExtractTransient)
	_ = w.fail(txCtx(), ExtractCVPayload{OrgID: orgID.String(), CandidateID: cand.ID.String()}, rawCause)

	if len(repo.updates) == 0 {
		t.Fatal("failure state not persisted")
	}
	got := repo.updates[len(repo.updates)-1]
	want := "temporary_error — retry with POST /cvs/{id}/extract"
	if got.ErrorMessage != want {
		t.Fatalf("error_message = %q, want %q", got.ErrorMessage, want)
	}
}

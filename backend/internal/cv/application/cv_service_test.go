package application

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/intivai/backend/internal/cv/domain"
	"github.com/intivai/backend/internal/iam/application"
	iamdomain "github.com/intivai/backend/internal/iam/domain"
	shareddomain "github.com/intivai/backend/internal/shared/domain"
)

type stubStore struct {
	uploads []string
	deletes []string
	failUp  error
	failDel error
}

func (s *stubStore) Upload(ctx context.Context, path string, r io.Reader, size int64, ct string) error {
	if s.failUp != nil {
		return s.failUp
	}
	s.uploads = append(s.uploads, path)
	return nil
}
func (s *stubStore) Download(ctx context.Context, path string) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("")), nil
}
func (s *stubStore) Delete(ctx context.Context, path string) error {
	if s.failDel != nil {
		return s.failDel
	}
	s.deletes = append(s.deletes, path)
	return nil
}

type stubRepo struct {
	created    []*domain.Candidate
	deleted    []uuid.UUID
	failCreate error
}

func (s *stubRepo) Create(ctx context.Context, c *domain.Candidate) error {
	if s.failCreate != nil {
		return s.failCreate
	}
	s.created = append(s.created, c)
	return nil
}
func (s *stubRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Candidate, error) {
	for _, c := range s.created {
		if c.ID == id {
			return c, nil
		}
	}
	return nil, domain.ErrNotFound
}
func (s *stubRepo) List(ctx context.Context, orgID uuid.UUID) ([]*domain.Candidate, error) {
	return s.created, nil
}
func (s *stubRepo) Update(ctx context.Context, c *domain.Candidate) error { return nil }
func (s *stubRepo) Delete(ctx context.Context, id uuid.UUID) error {
	s.deleted = append(s.deleted, id)
	return nil
}

type stubEnqueuer struct {
	fail bool
}

func (s *stubEnqueuer) Enqueue(ctx context.Context, task string, payload any, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	if s.fail {
		return nil, errors.New("redis down")
	}
	return &asynq.TaskInfo{ID: "t1"}, nil
}

func actor() application.AuthContext {
	return application.AuthContext{OrgID: uuid.New(), Role: string(iamdomain.RoleAdmin)}
}

func TestUploadHappyPath(t *testing.T) {
	store := &stubStore{}
	repo := &stubRepo{}
	svc := NewCVService(repo, store, &stubEnqueuer{})

	res, err := svc.Upload(context.Background(), actor(), "Jane", "j@x.io", []byte("pdf"), "application/pdf")
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	if res.Status != domain.StatusParsing {
		t.Fatalf("status = %s", res.Status)
	}
	if len(store.uploads) != 1 || len(repo.created) != 1 {
		t.Fatalf("upload=%d create=%d", len(store.uploads), len(repo.created))
	}
}

func TestUploadCompensatesOnEnqueueFailure(t *testing.T) {
	store := &stubStore{}
	repo := &stubRepo{}
	svc := NewCVService(repo, store, &stubEnqueuer{fail: true})

	if _, err := svc.Upload(context.Background(), actor(), "Jane", "j@x.io", []byte("pdf"), "application/pdf"); err == nil {
		t.Fatal("expected queue failure error")
	}
	if len(repo.deleted) != 1 {
		t.Fatalf("candidate not deleted: %d", len(repo.deleted))
	}
	if len(store.deletes) != 1 {
		t.Fatalf("file not deleted: %d", len(store.deletes))
	}
}

func TestUploadCompensatesOnRepoFailure(t *testing.T) {
	store := &stubStore{}
	repo := &stubRepo{failCreate: errors.New("db down")}
	svc := NewCVService(repo, store, &stubEnqueuer{})

	if _, err := svc.Upload(context.Background(), actor(), "Jane", "j@x.io", []byte("pdf"), "application/pdf"); err == nil {
		t.Fatal("expected create failure error")
	}
	if len(store.deletes) != 1 {
		t.Fatalf("orphan file not cleaned: %d", len(store.deletes))
	}
}

func TestUploadRejectsMemberRole(t *testing.T) {
	svc := NewCVService(&stubRepo{}, &stubStore{}, &stubEnqueuer{})
	_, err := svc.Upload(context.Background(), application.AuthContext{OrgID: uuid.New(), Role: "member"}, "J", "j@x.io", []byte("p"), "application/pdf")
	if err == nil {
		t.Fatal("member role accepted upload")
	}
}

func TestReExtractGuards(t *testing.T) {
	repo := &stubRepo{}
	enq := &stubEnqueuer{}
	svc := NewCVService(repo, &stubStore{}, enq)
	act := actor()
	orgID := act.OrgID

	// Unknown candidate → not found.
	if _, err := svc.ReExtract(context.Background(), act, uuid.New()); err == nil {
		t.Fatal("unknown candidate not rejected")
	}

	// Extracted candidate → not retryable.
	done := &domain.Candidate{Entity: entity(uuid.New()), OrgID: orgID, Status: domain.StatusExtracted}
	repo.created = append(repo.created, done)
	if _, err := svc.ReExtract(context.Background(), act, done.ID); err == nil {
		t.Fatal("extracted candidate accepted for re-extract")
	}

	// failed_extract → retryable, enqueues.
	failed := &domain.Candidate{Entity: entity(uuid.New()), OrgID: orgID, Status: domain.StatusFailedExtract}
	repo.created = append(repo.created, failed)
	if _, err := svc.ReExtract(context.Background(), act, failed.ID); err != nil {
		t.Fatalf("failed candidate not retryable: %v", err)
	}
}

func TestGetAndListSummary(t *testing.T) {
	repo := &stubRepo{}
	svc := NewCVService(repo, &stubStore{}, &stubEnqueuer{})
	act := actor()
	orgID := act.OrgID

	c := &domain.Candidate{
		Entity:       entity(uuid.New()),
		OrgID:        orgID,
		Name:         "Jane",
		Email:        "j@x.io",
		CVRawText:    "secret raw",
		CVStructured: []byte(`{"skills":["Go"]}`),
		Status:       domain.StatusParsed,
		CVOCRMethod:  "pdfcpu",
		ErrorMessage: "",
	}
	repo.created = append(repo.created, c)

	detail, err := svc.Get(context.Background(), act, c.ID)
	if err != nil {
		t.Fatal(err)
	}
	if detail.CVRawText != "secret raw" || len(detail.CVStructured) == 0 {
		t.Fatal("detail must include raw + structured")
	}

	list, err := svc.List(context.Background(), act)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("list = %d", len(list))
	}
	if list[0].Name == "" {
		t.Fatal("list item missing")
	}

	// Cross-org access blocked.
	other := actor()
	if _, err := svc.Get(context.Background(), other, c.ID); err == nil {
		t.Fatal("cross-org read allowed")
	}
}

func entity(id uuid.UUID) shareddomain.Entity {
	return shareddomain.Entity{ID: id, CreatedAt: time.Now().UTC()}
}

package application

import (
	"context"
	"encoding/json"
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
	scrapp "github.com/intivai/backend/internal/screening/application"
	scrdomain "github.com/intivai/backend/internal/screening/domain"
	shareddomain "github.com/intivai/backend/internal/shared/domain"
	sharederr "github.com/intivai/backend/internal/shared/errors"
	"github.com/intivai/backend/pkg/db"
	"gorm.io/gorm"
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

func (s *stubRepo) ListByIDs(ctx context.Context, orgID uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]*domain.Candidate, error) {
	return map[uuid.UUID]*domain.Candidate{}, nil
}

func (s *stubRepo) ConfirmReview(ctx context.Context, token string, structured []byte, name, email string) (uuid.UUID, uuid.UUID, error) {
	return uuid.Nil, uuid.Nil, nil
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
func (s *stubRepo) GetByReviewToken(ctx context.Context, token string) (*domain.Candidate, error) {
	for _, c := range s.created {
		if c.ReviewToken != nil && *c.ReviewToken == token {
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

type stubAppRepo struct {
	apps []*scrdomain.Application
}

func (s *stubAppRepo) Create(ctx context.Context, app *scrdomain.Application) error { return nil }
func (s *stubAppRepo) GetByID(ctx context.Context, id uuid.UUID) (*scrdomain.Application, error) {
	for _, a := range s.apps {
		if a.ID == id {
			return a, nil
		}
	}
	return nil, scrdomain.ErrNotFound
}
func (s *stubAppRepo) GetByCandidateJob(ctx context.Context, orgID, candidateID, jobID uuid.UUID) (*scrdomain.Application, error) {
	return nil, scrdomain.ErrNotFound
}
func (s *stubAppRepo) List(ctx context.Context, orgID, jobID uuid.UUID) ([]*scrdomain.Application, error) {
	return nil, nil
}
func (s *stubAppRepo) ByCandidate(ctx context.Context, orgID, candidateID uuid.UUID) ([]*scrdomain.Application, error) {
	return s.apps, nil
}
func (s *stubAppRepo) Update(ctx context.Context, app *scrdomain.Application) error { return nil }
func (s *stubAppRepo) UpdateDecision(ctx context.Context, orgID, id uuid.UUID, stage *scrdomain.Stage, notes *string) error {
	return nil
}
func (s *stubAppRepo) ListByIDs(ctx context.Context, orgID uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]*scrdomain.Application, error) {
	return map[uuid.UUID]*scrdomain.Application{}, nil
}
func (s *stubAppRepo) ApplyWithDedupe(ctx context.Context, orgID, jobID uuid.UUID, name, email string) (uuid.UUID, bool, error) {
	return uuid.Nil, false, scrdomain.ErrNotFound
}
func (s *stubAppRepo) CountRecentByCandidateEmail(ctx context.Context, orgID uuid.UUID, email string, since time.Time) (int, error) {
	return 0, nil
}

type confirmRecRepo struct {
	token      string
	structured []byte
	name       string
	email      string
	orgID      uuid.UUID
	candID     uuid.UUID
}

func (s *confirmRecRepo) ConfirmReview(ctx context.Context, token string, structured []byte, name, email string) (uuid.UUID, uuid.UUID, error) {
	s.token, s.structured, s.name, s.email = token, structured, name, email
	if s.orgID == uuid.Nil {
		return uuid.Nil, uuid.Nil, nil
	}
	return s.orgID, s.candID, nil
}
func (s *confirmRecRepo) Create(ctx context.Context, c *domain.Candidate) error { return nil }
func (s *confirmRecRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Candidate, error) {
	return nil, domain.ErrNotFound
}
func (s *confirmRecRepo) ListByIDs(ctx context.Context, orgID uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]*domain.Candidate, error) {
	return map[uuid.UUID]*domain.Candidate{}, nil
}
func (s *confirmRecRepo) GetByReviewToken(ctx context.Context, token string) (*domain.Candidate, error) {
	return nil, domain.ErrNotFound
}
func (s *confirmRecRepo) List(ctx context.Context, orgID uuid.UUID) ([]*domain.Candidate, error) {
	return nil, nil
}
func (s *confirmRecRepo) Update(ctx context.Context, c *domain.Candidate) error { return nil }
func (s *confirmRecRepo) Delete(ctx context.Context, id uuid.UUID) error        { return nil }

type confirmEnq struct {
	count int
	tasks map[string]int
}

func (s *confirmEnq) Enqueue(ctx context.Context, task string, payload any, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	if s.tasks == nil {
		s.tasks = map[string]int{}
	}
	s.tasks[task]++
	s.count++
	return &asynq.TaskInfo{ID: "t1"}, nil
}

func TestConfirmProfilePassesNameEmailToRepo(t *testing.T) {
	repo := &confirmRecRepo{orgID: uuid.New(), candID: uuid.New()}
	svc := NewCVService(repo, &stubAppRepo{}, &stubStore{}, &confirmEnq{}, nil)

	// RunInTx reuses the tx carried by the context — the service pool (nil in
	// unit tests) is never touched (same contract as worker unit tests).
	ctx := db.WithTx(context.Background(), &gorm.DB{})
	structured := []byte(`{"skills":["Go"]}`)
	if err := svc.ConfirmProfile(ctx, "tok-1", " Jane ", "  j@x.io", structured); err != nil {
		t.Fatalf("confirm: %v", err)
	}
	if repo.token != "tok-1" || string(repo.structured) != string(structured) {
		t.Fatalf("repo got token=%q structured=%q", repo.token, repo.structured)
	}
	if repo.name != "Jane" || repo.email != "j@x.io" {
		t.Fatalf("repo got name=%q email=%q, want trimmed Jane/j@x.io", repo.name, repo.email)
	}
}

func TestConfirmProfileInvalidTokenNotFound(t *testing.T) {
	svc := NewCVService(&confirmRecRepo{}, &stubAppRepo{}, &stubStore{}, &confirmEnq{}, nil)

	err := svc.ConfirmProfile(context.Background(), "tok-bad", "Jane", "j@x.io", []byte(`{"skills":["Go"]}`))
	var nf *sharederr.NotFoundError
	if !errors.As(err, &nf) {
		t.Fatalf("want NotFoundError, got %v", err)
	}
}

func TestConfirmProfileValidationCaps(t *testing.T) {
	svc := NewCVService(&confirmRecRepo{orgID: uuid.New(), candID: uuid.New()}, &stubAppRepo{}, &stubStore{}, &confirmEnq{}, nil)

	long := strings.Repeat("x", 2001)
	cases := []struct {
		name    string
		cmd     func() error
		wantMsg string
	}{
		{"empty name", func() error {
			return svc.ConfirmProfile(context.Background(), "t", "", "j@x.io", []byte(`{}`))
		}, "name is required"},
		{"whitespace name", func() error {
			return svc.ConfirmProfile(context.Background(), "t", "   ", "j@x.io", []byte(`{}`))
		}, "name is required"},
		{"name too long", func() error {
			return svc.ConfirmProfile(context.Background(), "t", strings.Repeat("a", 201), "j@x.io", []byte(`{}`))
		}, "name"},
		{"invalid email", func() error {
			return svc.ConfirmProfile(context.Background(), "t", "Jane", "not-an-email", []byte(`{}`))
		}, "email"},
		{"email too long", func() error {
			return svc.ConfirmProfile(context.Background(), "t", "Jane", strings.Repeat("a", 250)+"@x.io", []byte(`{}`))
		}, "email"},
		{"too many skills", func() error {
			skills := make([]string, 51)
			for i := range skills {
				skills[i] = "s"
			}
			return svc.ConfirmProfile(context.Background(), "t", "Jane", "j@x.io", mustJSON(t, map[string]any{"skills": skills}))
		}, "skills"},
		{"skill too long", func() error {
			return svc.ConfirmProfile(context.Background(), "t", "Jane", "j@x.io", mustJSON(t, map[string]any{"skills": []string{long}}))
		}, "skill"},
		{"too many certs", func() error {
			certs := make([]string, 26)
			for i := range certs {
				certs[i] = "c"
			}
			return svc.ConfirmProfile(context.Background(), "t", "Jane", "j@x.io", mustJSON(t, map[string]any{"certifications": certs}))
		}, "certifications"},
		{"cert too long", func() error {
			return svc.ConfirmProfile(context.Background(), "t", "Jane", "j@x.io", mustJSON(t, map[string]any{"certifications": []string{long}}))
		}, "certification"},
		{"education too long", func() error {
			return svc.ConfirmProfile(context.Background(), "t", "Jane", "j@x.io", mustJSON(t, map[string]any{"education": long}))
		}, "education"},
		{"summary too long", func() error {
			return svc.ConfirmProfile(context.Background(), "t", "Jane", "j@x.io", mustJSON(t, map[string]any{"summary": long}))
		}, "summary"},
		{"experience negative", func() error {
			return svc.ConfirmProfile(context.Background(), "t", "Jane", "j@x.io", mustJSON(t, map[string]any{"experience_years": -1}))
		}, "experience"},
		{"experience too high", func() error {
			return svc.ConfirmProfile(context.Background(), "t", "Jane", "j@x.io", mustJSON(t, map[string]any{"experience_years": 51}))
		}, "experience"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.cmd()
			var de *sharederr.DomainError
			if !errors.As(err, &de) {
				t.Fatalf("want DomainError, got %v", err)
			}
			if de.Code != "CANDIDATE_PROFILE_INVALID" {
				t.Fatalf("code = %q, want CANDIDATE_PROFILE_INVALID", de.Code)
			}
			if !strings.Contains(de.Message, tc.wantMsg) {
				t.Fatalf("message %q does not mention %q", de.Message, tc.wantMsg)
			}
		})
	}
}

func TestConfirmProfileEmailOptional(t *testing.T) {
	repo := &confirmRecRepo{orgID: uuid.New(), candID: uuid.New()}
	svc := NewCVService(repo, &stubAppRepo{}, &stubStore{}, &confirmEnq{}, nil)

	// Bulk-uploaded candidates may have no email — empty email must pass.
	if err := svc.ConfirmProfile(db.WithTx(context.Background(), &gorm.DB{}), "t", "Jane", "", []byte(`{}`)); err != nil {
		t.Fatalf("empty email rejected: %v", err)
	}
	if repo.email != "" {
		t.Fatalf("email = %q, want empty", repo.email)
	}
}

func TestConfirmProfileEnqueuesScorePerApplication(t *testing.T) {
	orgID := uuid.New()
	candID := uuid.New()
	apps := []*scrdomain.Application{
		scrdomain.NewApplication(orgID, candID, uuid.New()),
		scrdomain.NewApplication(orgID, candID, uuid.New()),
	}
	enq := &confirmEnq{}
	svc := NewCVService(&confirmRecRepo{orgID: orgID, candID: candID}, &stubAppRepo{apps: apps}, &stubStore{}, enq, nil)

	if err := svc.ConfirmProfile(db.WithTx(context.Background(), &gorm.DB{}), "tok", "Jane", "j@x.io", []byte(`{"skills":["Go"]}`)); err != nil {
		t.Fatalf("confirm: %v", err)
	}
	if enq.tasks[scrapp.TaskScoreCV] != 2 {
		t.Fatalf("enqueued %d score_cv tasks, want 2: %+v", enq.tasks[scrapp.TaskScoreCV], enq.tasks)
	}
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func actor() application.AuthContext {
	return application.AuthContext{OrgID: uuid.New(), Role: string(iamdomain.RoleAdmin)}
}

func TestUploadHappyPath(t *testing.T) {
	store := &stubStore{}
	repo := &stubRepo{}
	svc := NewCVService(repo, nil, store, &stubEnqueuer{}, nil)

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
	svc := NewCVService(repo, nil, store, &stubEnqueuer{fail: true}, nil)

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
	svc := NewCVService(repo, nil, store, &stubEnqueuer{}, nil)

	if _, err := svc.Upload(context.Background(), actor(), "Jane", "j@x.io", []byte("pdf"), "application/pdf"); err == nil {
		t.Fatal("expected create failure error")
	}
	if len(store.deletes) != 1 {
		t.Fatalf("orphan file not cleaned: %d", len(store.deletes))
	}
}

func TestUploadRejectsMemberRole(t *testing.T) {
	svc := NewCVService(&stubRepo{}, nil, &stubStore{}, &stubEnqueuer{}, nil)
	_, err := svc.Upload(context.Background(), application.AuthContext{OrgID: uuid.New(), Role: "member"}, "J", "j@x.io", []byte("p"), "application/pdf")
	if err == nil {
		t.Fatal("member role accepted upload")
	}
}

func TestReExtractGuards(t *testing.T) {
	repo := &stubRepo{}
	enq := &stubEnqueuer{}
	svc := NewCVService(repo, nil, &stubStore{}, enq, nil)
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

	// failed_ocr → retryable, enqueues parse task.
	failedOcr := &domain.Candidate{Entity: entity(uuid.New()), OrgID: orgID, Status: domain.StatusFailedOCR}
	repo.created = append(repo.created, failedOcr)
	if res, err := svc.ReExtract(context.Background(), act, failedOcr.ID); err != nil || res.Status != domain.StatusParsing {
		t.Fatalf("failed_ocr candidate not retryable to parsing: %v", err)
	}
}

func TestGetAndListSummary(t *testing.T) {
	repo := &stubRepo{}
	svc := NewCVService(repo, nil, &stubStore{}, &stubEnqueuer{}, nil)
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

func TestDeleteCandidate(t *testing.T) {
	repo := &stubRepo{}
	store := &stubStore{}
	svc := NewCVService(repo, nil, store, &stubEnqueuer{}, nil)
	act := actor()

	c := &domain.Candidate{
		Entity: entity(uuid.New()),
		OrgID:  act.OrgID,
		Name:   "Jane",
		Email:  "j@x.io",
		CVPath: "cvs/org/c1.pdf",
	}
	repo.created = append(repo.created, c)

	// Role check
	member := application.AuthContext{OrgID: act.OrgID, Role: "member"}
	if err := svc.DeleteCandidate(context.Background(), member, c.ID); err == nil {
		t.Fatal("expected error for member role")
	}

	// Not found
	if err := svc.DeleteCandidate(context.Background(), act, uuid.New()); err == nil {
		t.Fatal("expected not found error")
	}

	// Cross org
	other := actor()
	if err := svc.DeleteCandidate(context.Background(), other, c.ID); err == nil {
		t.Fatal("expected forbidden error for cross org")
	}

	// Happy path
	if err := svc.DeleteCandidate(context.Background(), act, c.ID); err != nil {
		t.Fatalf("delete candidate failed: %v", err)
	}
	if len(store.deletes) != 1 {
		t.Fatalf("expected file delete, got %d", len(store.deletes))
	}
}

func TestPayloadUUIDHelper(t *testing.T) {
	valid := uuid.New().String()
	u, err := payloadUUID(valid)
	if err != nil || u.String() != valid {
		t.Fatalf("expected valid uuid, got %v, %v", u, err)
	}
	if _, err := payloadUUID("invalid"); err == nil {
		t.Fatal("expected error on invalid uuid")
	}
}

func TestReExtract(t *testing.T) {
	repo := &stubRepo{}
	store := &stubStore{}
	enq := &stubEnqueuer{}
	svc := NewCVService(repo, nil, store, enq, nil)
	act := actor()

	c := &domain.Candidate{
		Entity: entity(uuid.New()),
		OrgID:  act.OrgID,
		Name:   "Jane",
		Email:  "j@x.io",
		Status: domain.StatusFailedExtract,
	}
	repo.created = append(repo.created, c)

	// Unauthorized role
	member := application.AuthContext{OrgID: act.OrgID, Role: "member"}
	if _, err := svc.ReExtract(context.Background(), member, c.ID); err == nil {
		t.Fatal("expected error for member role")
	}

	// Not found
	if _, err := svc.ReExtract(context.Background(), act, uuid.New()); err == nil {
		t.Fatal("expected not found error")
	}

	// Success for FailedExtract
	res, err := svc.ReExtract(context.Background(), act, c.ID)
	if err != nil || res.Status != domain.StatusExtracting {
		t.Fatalf("expected extracting status, got %v, %v", res, err)
	}

	// Success for FailedOCR
	c.Status = domain.StatusFailedOCR
	res2, err := svc.ReExtract(context.Background(), act, c.ID)
	if err != nil || res2.Status != domain.StatusParsing {
		t.Fatalf("expected parsing status, got %v, %v", res2, err)
	}
}

func entity(id uuid.UUID) shareddomain.Entity {
	return shareddomain.Entity{ID: id, CreatedAt: time.Now().UTC()}
}

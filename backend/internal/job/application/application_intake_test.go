package application_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	cvapp "github.com/intivai/backend/internal/cv/application"
	cvdomain "github.com/intivai/backend/internal/cv/domain"
	jobapp "github.com/intivai/backend/internal/job/application"
	jobdomain "github.com/intivai/backend/internal/job/domain"
	notifapp "github.com/intivai/backend/internal/notification/application"
	scrdomain "github.com/intivai/backend/internal/screening/domain"
	sharederr "github.com/intivai/backend/internal/shared/errors"
	"github.com/rs/zerolog"
)

type mockJobReader struct {
	job *jobapp.PublicJobInfo
	err error
}

func (m *mockJobReader) GetPublicDetail(_ context.Context, _ uuid.UUID) (*jobapp.PublicJobInfo, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.job, nil
}

type mockCandidateRepo struct {
	deletedIDs []uuid.UUID
}

func (m *mockCandidateRepo) Create(_ context.Context, _ *cvdomain.Candidate) error { return nil }
func (m *mockCandidateRepo) GetByID(_ context.Context, _ uuid.UUID) (*cvdomain.Candidate, error) {
	return nil, nil
}
func (m *mockCandidateRepo) ListByIDs(_ context.Context, _ uuid.UUID, _ []uuid.UUID) (map[uuid.UUID]*cvdomain.Candidate, error) {
	return nil, nil
}
func (m *mockCandidateRepo) GetByReviewToken(_ context.Context, _ string) (*cvdomain.Candidate, error) {
	return nil, nil
}
func (m *mockCandidateRepo) ConfirmReview(_ context.Context, _ string, _ []byte, _, _ string) (uuid.UUID, uuid.UUID, error) {
	return uuid.Nil, uuid.Nil, nil
}
func (m *mockCandidateRepo) List(_ context.Context, _ uuid.UUID) ([]*cvdomain.Candidate, error) {
	return nil, nil
}
func (m *mockCandidateRepo) Update(_ context.Context, _ *cvdomain.Candidate) error { return nil }
func (m *mockCandidateRepo) Delete(_ context.Context, id uuid.UUID) error {
	m.deletedIDs = append(m.deletedIDs, id)
	return nil
}

type mockApplicationRepo struct {
	recentCount int
	candidateID uuid.UUID
	isNew       bool
	applyErr    error
}

func (m *mockApplicationRepo) Create(_ context.Context, _ *scrdomain.Application) error { return nil }
func (m *mockApplicationRepo) GetByID(_ context.Context, _ uuid.UUID) (*scrdomain.Application, error) {
	return nil, nil
}
func (m *mockApplicationRepo) GetByCandidateJob(_ context.Context, _, _, _ uuid.UUID) (*scrdomain.Application, error) {
	return nil, nil
}
func (m *mockApplicationRepo) List(_ context.Context, _, _ uuid.UUID) ([]*scrdomain.Application, error) {
	return nil, nil
}
func (m *mockApplicationRepo) ByCandidate(_ context.Context, _, _ uuid.UUID) ([]*scrdomain.Application, error) {
	return nil, nil
}
func (m *mockApplicationRepo) Update(_ context.Context, _ *scrdomain.Application) error { return nil }
func (m *mockApplicationRepo) UpdateDecision(_ context.Context, _, _ uuid.UUID, _ *scrdomain.Stage, _ *string) error {
	return nil
}
func (m *mockApplicationRepo) ListByIDs(_ context.Context, _ uuid.UUID, _ []uuid.UUID) (map[uuid.UUID]*scrdomain.Application, error) {
	return nil, nil
}
func (m *mockApplicationRepo) ApplyWithDedupe(_ context.Context, _, _ uuid.UUID, _, _ string) (uuid.UUID, bool, error) {
	if m.applyErr != nil {
		return uuid.Nil, false, m.applyErr
	}
	return m.candidateID, m.isNew, nil
}
func (m *mockApplicationRepo) CountRecentByCandidateEmail(_ context.Context, _ uuid.UUID, _ string, _ time.Time) (int, error) {
	return m.recentCount, nil
}

type mockObjectStore struct {
	uploaded   map[string][]byte
	deleted    []string
	failUpload bool
}

func (m *mockObjectStore) Upload(_ context.Context, key string, r io.Reader, _ int64, _ string) error {
	if m.failUpload {
		return errors.New("s3 upload failed")
	}
	data, _ := io.ReadAll(r)
	if m.uploaded == nil {
		m.uploaded = make(map[string][]byte)
	}
	m.uploaded[key] = data
	return nil
}

func (m *mockObjectStore) Delete(_ context.Context, key string) error {
	m.deleted = append(m.deleted, key)
	return nil
}

type mockEnqueuer struct {
	tasks      []string
	failOnTask string
}

func (m *mockEnqueuer) Enqueue(_ context.Context, jobType string, _ any, _ ...asynq.Option) (*asynq.TaskInfo, error) {
	if m.failOnTask == jobType {
		return nil, errors.New("enqueue error")
	}
	m.tasks = append(m.tasks, jobType)
	return &asynq.TaskInfo{}, nil
}

type mockPortalRepo struct {
	magicTokens map[string]string
}

func (m *mockPortalRepo) CreateMagicToken(_ context.Context, email, token string, _ time.Time) error {
	if m.magicTokens == nil {
		m.magicTokens = make(map[string]string)
	}
	m.magicTokens[email] = token
	return nil
}
func (m *mockPortalRepo) CreateOTP(_ context.Context, _, _, _ string, _ time.Time) error {
	return nil
}
func (m *mockPortalRepo) LastRequestAt(_ context.Context, _ string) (*time.Time, error) {
	return nil, nil
}
func (m *mockPortalRepo) OTPCountSince(_ context.Context, _ string, _ time.Time) (int, error) {
	return 0, nil
}
func (m *mockPortalRepo) FindValidByToken(_ context.Context, _ string) (*scrdomain.CandidateOTP, error) {
	return nil, nil
}
func (m *mockPortalRepo) FindValidByCodeHash(_ context.Context, _, _ string) (*scrdomain.CandidateOTP, error) {
	return nil, nil
}
func (m *mockPortalRepo) IncrementAttempts(_ context.Context, _ string) error {
	return nil
}
func (m *mockPortalRepo) Consume(_ context.Context, _ uuid.UUID) (bool, error) {
	return true, nil
}
func (m *mockPortalRepo) PurgeExpired(_ context.Context, _ string) error {
	return nil
}
func (m *mockPortalRepo) EraseCandidate(_ context.Context, _ string) error {
	return nil
}

func TestApplicationIntake_NewCandidate_Success(t *testing.T) {
	orgID := uuid.New()
	jobID := uuid.New()
	candID := uuid.New()

	jobReader := &mockJobReader{
		job: &jobapp.PublicJobInfo{
			OrgID: orgID,
			Title: "Senior Go Engineer",
		},
	}
	candRepo := &mockCandidateRepo{}
	appRepo := &mockApplicationRepo{
		candidateID: candID,
		isNew:       true,
		recentCount: 0,
	}
	store := &mockObjectStore{}
	queue := &mockEnqueuer{}
	portalRepo := &mockPortalRepo{}
	logger := zerolog.Nop()

	service := jobapp.NewApplicationIntakeService(
		nil, jobReader, candRepo, appRepo, store, queue, portalRepo, "http://localhost:5173", logger,
	)

	pdfData := []byte("%PDF-1.4 minimal valid header for test")
	cmd := jobapp.ApplyCommand{
		JobID:       jobID,
		Name:        "Alice Applicant",
		Email:       "alice@example.com",
		Resume:      bytes.NewReader(pdfData),
		ResumeSize:  int64(len(pdfData)),
		ContentType: "application/pdf",
	}

	res, err := service.Apply(context.Background(), cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.CandidateID != candID {
		t.Errorf("expected candidate ID %s, got %s", candID, res.CandidateID)
	}
	if res.Status != "submitted" {
		t.Errorf("expected status 'submitted', got %s", res.Status)
	}

	expectedKey := "cvs/" + orgID.String() + "/" + candID.String() + ".pdf"
	if _, ok := store.uploaded[expectedKey]; !ok {
		t.Errorf("expected upload at key %s, but not found", expectedKey)
	}

	var hasParse, hasConfirm, hasPortal bool
	for _, task := range queue.tasks {
		if task == cvapp.TaskParseCV {
			hasParse = true
		}
		if task == notifapp.TaskSendEmail {
			if !hasConfirm {
				hasConfirm = true
			} else {
				hasPortal = true
			}
		}
	}
	if !hasParse {
		t.Errorf("TaskParseCV was not enqueued")
	}
	if !hasConfirm {
		t.Errorf("Confirmation TaskSendEmail was not enqueued")
	}
	if !hasPortal {
		t.Errorf("Portal access TaskSendEmail was not enqueued")
	}
}

func TestApplicationIntake_InvalidPDF_ReturnsError(t *testing.T) {
	service := jobapp.NewApplicationIntakeService(
		nil, &mockJobReader{}, &mockCandidateRepo{}, &mockApplicationRepo{}, &mockObjectStore{}, &mockEnqueuer{}, &mockPortalRepo{}, "http://localhost:5173", zerolog.Nop(),
	)

	cmd := jobapp.ApplyCommand{
		JobID:       uuid.New(),
		Name:        "Bob",
		Email:       "bob@example.com",
		Resume:      strings.NewReader("not a pdf"),
		ResumeSize:  9,
		ContentType: "application/pdf",
	}

	_, err := service.Apply(context.Background(), cmd)
	if err == nil {
		t.Fatal("expected error for non-PDF, got nil")
	}
	var de *sharederr.DomainError
	if !errors.As(err, &de) || de.Code != "INVALID_INPUT" {
		t.Fatalf("expected INVALID_INPUT domain error, got %v", err)
	}
}

func TestApplicationIntake_JobNotFound_ReturnsError(t *testing.T) {
	service := jobapp.NewApplicationIntakeService(
		nil, &mockJobReader{err: jobdomain.ErrNotFound}, &mockCandidateRepo{}, &mockApplicationRepo{}, &mockObjectStore{}, &mockEnqueuer{}, &mockPortalRepo{}, "http://localhost:5173", zerolog.Nop(),
	)

	pdfData := []byte("%PDF-1.4 test")
	cmd := jobapp.ApplyCommand{
		JobID:       uuid.New(),
		Name:        "Bob",
		Email:       "bob@example.com",
		Resume:      bytes.NewReader(pdfData),
		ResumeSize:  int64(len(pdfData)),
		ContentType: "application/pdf",
	}

	_, err := service.Apply(context.Background(), cmd)
	if err == nil {
		t.Fatal("expected error for not found job, got nil")
	}
	var nf *sharederr.NotFoundError
	if !errors.As(err, &nf) {
		t.Fatalf("expected NotFoundError, got %v", err)
	}
}

func TestApplicationIntake_AbuseCapExceeded_ReturnsError(t *testing.T) {
	appRepo := &mockApplicationRepo{
		recentCount: 10,
	}
	service := jobapp.NewApplicationIntakeService(
		nil, &mockJobReader{job: &jobapp.PublicJobInfo{OrgID: uuid.New()}}, &mockCandidateRepo{}, appRepo, &mockObjectStore{}, &mockEnqueuer{}, &mockPortalRepo{}, "http://localhost:5173", zerolog.Nop(),
	)

	pdfData := []byte("%PDF-1.4 test")
	cmd := jobapp.ApplyCommand{
		JobID:       uuid.New(),
		Name:        "Spammer",
		Email:       "spam@example.com",
		Resume:      bytes.NewReader(pdfData),
		ResumeSize:  int64(len(pdfData)),
		ContentType: "application/pdf",
	}

	_, err := service.Apply(context.Background(), cmd)
	if err == nil {
		t.Fatal("expected error for abuse cap exceeded, got nil")
	}
	var de *sharederr.DomainError
	if !errors.As(err, &de) || de.Code != "TOO_MANY_REQUESTS" {
		t.Fatalf("expected TOO_MANY_REQUESTS domain error, got %v", err)
	}
}

func TestApplicationIntake_StorageUploadFailure_RollsBackNewCandidate(t *testing.T) {
	candID := uuid.New()
	candRepo := &mockCandidateRepo{}
	appRepo := &mockApplicationRepo{
		candidateID: candID,
		isNew:       true,
	}
	store := &mockObjectStore{failUpload: true}

	service := jobapp.NewApplicationIntakeService(
		nil, &mockJobReader{job: &jobapp.PublicJobInfo{OrgID: uuid.New()}}, candRepo, appRepo, store, &mockEnqueuer{}, &mockPortalRepo{}, "http://localhost:5173", zerolog.Nop(),
	)

	pdfData := []byte("%PDF-1.4 test")
	cmd := jobapp.ApplyCommand{
		JobID:       uuid.New(),
		Name:        "Charlie",
		Email:       "charlie@example.com",
		Resume:      bytes.NewReader(pdfData),
		ResumeSize:  int64(len(pdfData)),
		ContentType: "application/pdf",
	}

	_, err := service.Apply(context.Background(), cmd)
	if err == nil {
		t.Fatal("expected error on storage failure, got nil")
	}

	if len(candRepo.deletedIDs) != 1 || candRepo.deletedIDs[0] != candID {
		t.Fatalf("expected candidate %s to be rolled back/deleted, got %v", candID, candRepo.deletedIDs)
	}
}

func TestApplicationIntake_QueueFailure_RollsBackStorageAndCandidate(t *testing.T) {
	candID := uuid.New()
	orgID := uuid.New()
	candRepo := &mockCandidateRepo{}
	appRepo := &mockApplicationRepo{
		candidateID: candID,
		isNew:       true,
	}
	store := &mockObjectStore{}
	queue := &mockEnqueuer{failOnTask: cvapp.TaskParseCV}

	service := jobapp.NewApplicationIntakeService(
		nil, &mockJobReader{job: &jobapp.PublicJobInfo{OrgID: orgID}}, candRepo, appRepo, store, queue, &mockPortalRepo{}, "http://localhost:5173", zerolog.Nop(),
	)

	pdfData := []byte("%PDF-1.4 test")
	cmd := jobapp.ApplyCommand{
		JobID:       uuid.New(),
		Name:        "Dana",
		Email:       "dana@example.com",
		Resume:      bytes.NewReader(pdfData),
		ResumeSize:  int64(len(pdfData)),
		ContentType: "application/pdf",
	}

	_, err := service.Apply(context.Background(), cmd)
	if err == nil {
		t.Fatal("expected error on queue failure, got nil")
	}

	expectedKey := "cvs/" + orgID.String() + "/" + candID.String() + ".pdf"
	if len(store.deleted) != 1 || store.deleted[0] != expectedKey {
		t.Fatalf("expected storage file %s to be deleted on failure, got %v", expectedKey, store.deleted)
	}

	if len(candRepo.deletedIDs) != 1 || candRepo.deletedIDs[0] != candID {
		t.Fatalf("expected candidate %s to be deleted on failure, got %v", candID, candRepo.deletedIDs)
	}
}

func TestApplicationIntake_Reapply_UploadFailure_DoesNotDeleteOldCandidate(t *testing.T) {
	candID := uuid.New()
	candRepo := &mockCandidateRepo{}
	appRepo := &mockApplicationRepo{
		candidateID: candID,
		isNew:       false, // Existing candidate
	}
	store := &mockObjectStore{failUpload: true}

	service := jobapp.NewApplicationIntakeService(
		nil, &mockJobReader{job: &jobapp.PublicJobInfo{OrgID: uuid.New()}}, candRepo, appRepo, store, &mockEnqueuer{}, &mockPortalRepo{}, "http://localhost:5173", zerolog.Nop(),
	)

	pdfData := []byte("%PDF-1.4 test")
	cmd := jobapp.ApplyCommand{
		JobID:       uuid.New(),
		Name:        "Eve",
		Email:       "eve@example.com",
		Resume:      bytes.NewReader(pdfData),
		ResumeSize:  int64(len(pdfData)),
		ContentType: "application/pdf",
	}

	_, err := service.Apply(context.Background(), cmd)
	if err == nil {
		t.Fatal("expected error on storage upload failure, got nil")
	}

	if len(candRepo.deletedIDs) != 0 {
		t.Fatalf("expected 0 candidate deletions for existing candidate, got %v", candRepo.deletedIDs)
	}
}

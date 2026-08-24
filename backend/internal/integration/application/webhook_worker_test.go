package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/intivai/backend/internal/integration/domain"
	shareddomain "github.com/intivai/backend/internal/shared/domain"
	"github.com/intivai/backend/pkg/db"
	"github.com/rs/zerolog"
	"gorm.io/gorm"
)

type mockWebhookRepo struct {
	deliveries map[uuid.UUID]*domain.WebhookDelivery
	configs    map[uuid.UUID]*domain.WebhookConfig
}

func newMockRepo() *mockWebhookRepo {
	return &mockWebhookRepo{
		deliveries: make(map[uuid.UUID]*domain.WebhookDelivery),
		configs:    make(map[uuid.UUID]*domain.WebhookConfig),
	}
}

func (m *mockWebhookRepo) CreateConfig(_ context.Context, _ *domain.WebhookConfig) error { return nil }
func (m *mockWebhookRepo) GetConfigByID(_ context.Context, id uuid.UUID) (*domain.WebhookConfig, error) {
	cfg, ok := m.configs[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return cfg, nil
}
func (m *mockWebhookRepo) ListConfigsByOrg(_ context.Context, _ uuid.UUID) ([]*domain.WebhookConfig, error) {
	return nil, nil
}
func (m *mockWebhookRepo) UpdateConfig(_ context.Context, _ *domain.WebhookConfig) error { return nil }
func (m *mockWebhookRepo) DeleteConfig(_ context.Context, _ uuid.UUID) error             { return nil }
func (m *mockWebhookRepo) CreateDelivery(_ context.Context, _ *domain.WebhookDelivery) error {
	return nil
}
func (m *mockWebhookRepo) GetDeliveryByID(_ context.Context, id uuid.UUID) (*domain.WebhookDelivery, error) {
	d, ok := m.deliveries[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	copy := *d
	copy.Payload = append([]byte(nil), d.Payload...)
	return &copy, nil
}
func (m *mockWebhookRepo) IncrementDeliveryAttempt(_ context.Context, id uuid.UUID) (int, bool, error) {
	delivery, ok := m.deliveries[id]
	if !ok || delivery.FinalStatus != "pending" || delivery.Attempts >= maxDeliveryAttempts {
		return 0, false, nil
	}
	delivery.Attempts++
	return delivery.Attempts, true, nil
}
func (m *mockWebhookRepo) ListDeliveriesByWebhook(_ context.Context, _ uuid.UUID, _ int) ([]*domain.WebhookDelivery, error) {
	return nil, nil
}
func (m *mockWebhookRepo) UpdateDelivery(_ context.Context, d *domain.WebhookDelivery) error {
	copy := *d
	copy.Payload = append([]byte(nil), d.Payload...)
	m.deliveries[d.ID] = &copy
	return nil
}
func (m *mockWebhookRepo) UpdateDeliveryIfNotDelivered(ctx context.Context, d *domain.WebhookDelivery) (bool, error) {
	current, ok := m.deliveries[d.ID]
	if !ok || current.FinalStatus == "delivered" {
		return false, nil
	}
	return true, m.UpdateDelivery(ctx, d)
}
func (m *mockWebhookRepo) ListConfigsByEvent(_ context.Context, _ uuid.UUID, _ domain.WebhookEvent) ([]*domain.WebhookConfig, error) {
	return nil, nil
}

// txCtx carries a pre-set transaction so handle()'s RunInTx short-circuits —
// unit tests exercise the retry/state logic without a real pool.
func txCtx() context.Context {
	return db.WithTx(context.Background(), &gorm.DB{})
}

func newWorker(repo *mockWebhookRepo) *WebhookWorker {
	return NewWebhookWorker(repo, nil, zerolog.Nop())
}

func TestWebhookWorkerValidDelivery(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Intivai-Signature") == "" {
			t.Error("missing signature header")
		}
		if r.Header.Get("X-Intivai-Event") != "interview.completed" {
			t.Error("missing or wrong event header")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{"ok":true}`)
	}))
	defer srv.Close()

	repo := newMockRepo()
	cfgID := uuid.New()
	deliveryID := uuid.New()
	orgID := uuid.New()

	cfg := &domain.WebhookConfig{
		Entity: shareddomain.Entity{ID: cfgID},
		URL:    srv.URL,
		Secret: "test-secret",
	}
	repo.configs[cfgID] = cfg

	payload, _ := json.Marshal(map[string]string{"event": "interview.completed"})
	delivery := &domain.WebhookDelivery{
		Entity:      shareddomain.Entity{ID: deliveryID},
		OrgID:       orgID,
		WebhookID:   cfgID,
		Event:       domain.EventInterviewCompleted,
		Payload:     payload,
		FinalStatus: "pending",
	}
	repo.deliveries[deliveryID] = delivery

	worker := newWorker(repo)
	taskPayload, _ := json.Marshal(DeliverWebhookPayload{
		DeliveryID: deliveryID.String(),
		OrgID:      orgID.String(),
	})
	task := asynq.NewTask(TaskDeliverWebhook, taskPayload)

	if err := worker.handle(txCtx(), task); err != nil {
		t.Fatalf("worker returned error: %v", err)
	}

	got := repo.deliveries[deliveryID]
	if got.FinalStatus != "delivered" {
		t.Fatalf("want final_status=delivered, got %q", got.FinalStatus)
	}
	if got.StatusCode != 200 {
		t.Fatalf("want status_code=200, got %d", got.StatusCode)
	}
	if got.Attempts != 1 {
		t.Fatalf("want attempts=1, got %d", got.Attempts)
	}
}

func TestWebhookWorkerInvalidPayload(t *testing.T) {
	repo := newMockRepo()
	worker := newWorker(repo)

	task := asynq.NewTask(TaskDeliverWebhook, []byte("not-json"))
	err := worker.handle(txCtx(), task)
	if !errors.Is(err, asynq.SkipRetry) {
		t.Fatalf("want asynq.SkipRetry, got %v", err)
	}
}

func TestWebhookWorkerConfigNotFound(t *testing.T) {
	repo := newMockRepo()
	deliveryID := uuid.New()
	cfgID := uuid.New()
	orgID := uuid.New()

	delivery := &domain.WebhookDelivery{
		Entity:      shareddomain.Entity{ID: deliveryID},
		OrgID:       orgID,
		WebhookID:   cfgID,
		Event:       domain.EventInterviewCompleted,
		Payload:     []byte(`{}`),
		FinalStatus: "pending",
	}
	repo.deliveries[deliveryID] = delivery

	worker := newWorker(repo)
	taskPayload, _ := json.Marshal(DeliverWebhookPayload{
		DeliveryID: deliveryID.String(),
		OrgID:      orgID.String(),
	})
	task := asynq.NewTask(TaskDeliverWebhook, taskPayload)

	err := worker.handle(txCtx(), task)
	if !errors.Is(err, asynq.SkipRetry) {
		t.Fatalf("want asynq.SkipRetry, got %v", err)
	}
}

func TestWebhookWorkerDeliveryNotFound(t *testing.T) {
	repo := newMockRepo()
	worker := newWorker(repo)
	taskPayload, _ := json.Marshal(DeliverWebhookPayload{
		DeliveryID: uuid.NewString(),
		OrgID:      uuid.NewString(),
	})
	err := worker.handle(txCtx(), asynq.NewTask(TaskDeliverWebhook, taskPayload))
	if !errors.Is(err, asynq.SkipRetry) {
		t.Fatalf("want asynq.SkipRetry, got %v", err)
	}
}

// RED: 5xx must return an error (asynq retries) and persist the attempt; the
// attempt counter survives the retry because the state is written BEFORE the
// error is returned.
func TestWebhookWorkerTransientFailureRetriesThenFails(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	repo := newMockRepo()
	cfgID := uuid.New()
	deliveryID := uuid.New()
	orgID := uuid.New()
	repo.configs[cfgID] = &domain.WebhookConfig{Entity: shareddomain.Entity{ID: cfgID}, URL: srv.URL, Secret: "s"}
	repo.deliveries[deliveryID] = &domain.WebhookDelivery{
		Entity:      shareddomain.Entity{ID: deliveryID},
		OrgID:       orgID,
		WebhookID:   cfgID,
		Event:       domain.EventInterviewCompleted,
		Payload:     []byte(`{}`),
		FinalStatus: "pending",
	}

	worker := newWorker(repo)
	taskPayload, _ := json.Marshal(DeliverWebhookPayload{DeliveryID: deliveryID.String(), OrgID: orgID.String()})
	task := asynq.NewTask(TaskDeliverWebhook, taskPayload)

	// Attempt 1: transient → error (asynq retry).
	err := worker.handle(txCtx(), task)
	if err == nil || errors.Is(err, asynq.SkipRetry) {
		t.Fatalf("attempt 1 want retryable error, got %v", err)
	}
	got := repo.deliveries[deliveryID]
	if got.FinalStatus != "pending" {
		t.Fatalf("attempt 1 want final_status=pending, got %q", got.FinalStatus)
	}
	if got.Attempts != 1 {
		t.Fatalf("attempt 1 want attempts=1, got %d", got.Attempts)
	}

	// Attempts 2, 3: same.
	for i := 2; i <= 3; i++ {
		err := worker.handle(txCtx(), task)
		if i < 3 {
			if err == nil || errors.Is(err, asynq.SkipRetry) {
				t.Fatalf("attempt %d want retryable error, got %v", i, err)
			}
		} else {
			if !errors.Is(err, asynq.SkipRetry) {
				t.Fatalf("attempt %d want asynq.SkipRetry, got %v", i, err)
			}
		}
	}
	if calls != 3 {
		t.Fatalf("want 3 HTTP calls, got %d", calls)
	}
	got = repo.deliveries[deliveryID]
	if got.FinalStatus != "failed" {
		t.Fatalf("want final_status=failed after retries, got %q", got.FinalStatus)
	}
	if got.Attempts != 3 {
		t.Fatalf("want attempts=3, got %d", got.Attempts)
	}
}

// 4xx is permanent: first attempt marks failed and returns SkipRetry — no
// pointless retry loop.
func TestWebhookWorkerClientErrorIsPermanent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	repo := newMockRepo()
	cfgID := uuid.New()
	deliveryID := uuid.New()
	orgID := uuid.New()
	repo.configs[cfgID] = &domain.WebhookConfig{Entity: shareddomain.Entity{ID: cfgID}, URL: srv.URL, Secret: "s"}
	repo.deliveries[deliveryID] = &domain.WebhookDelivery{
		Entity:      shareddomain.Entity{ID: deliveryID},
		OrgID:       orgID,
		WebhookID:   cfgID,
		Event:       domain.EventInterviewCompleted,
		Payload:     []byte(`{}`),
		FinalStatus: "pending",
	}

	worker := newWorker(repo)
	taskPayload, _ := json.Marshal(DeliverWebhookPayload{DeliveryID: deliveryID.String(), OrgID: orgID.String()})
	err := worker.handle(txCtx(), asynq.NewTask(TaskDeliverWebhook, taskPayload))
	if !errors.Is(err, asynq.SkipRetry) {
		t.Fatalf("want asynq.SkipRetry, got %v", err)
	}
	if got := repo.deliveries[deliveryID]; got.FinalStatus != "failed" {
		t.Fatalf("want final_status=failed, got %q", got.FinalStatus)
	}
}

func TestWebhookWorkerRejectsRedirectToPrivateAddress(t *testing.T) {
	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://127.0.0.1/secret", http.StatusFound)
	}))
	defer redirect.Close()

	repo := newMockRepo()
	cfgID, deliveryID, orgID := uuid.New(), uuid.New(), uuid.New()
	repo.configs[cfgID] = &domain.WebhookConfig{
		Entity: shareddomain.Entity{ID: cfgID}, URL: redirect.URL, Secret: "s",
	}
	repo.deliveries[deliveryID] = &domain.WebhookDelivery{
		Entity: shareddomain.Entity{ID: deliveryID}, OrgID: orgID, WebhookID: cfgID,
		Event: domain.EventInterviewCompleted, Payload: []byte(`{}`), FinalStatus: "pending",
	}

	worker := newWorker(repo)
	taskPayload, _ := json.Marshal(DeliverWebhookPayload{DeliveryID: deliveryID.String(), OrgID: orgID.String()})
	err := worker.handle(txCtx(), asynq.NewTask(TaskDeliverWebhook, taskPayload))
	if err == nil || errors.Is(err, asynq.SkipRetry) {
		t.Fatalf("redirect rejection should be retryable delivery failure, got %v", err)
	}
	if repo.deliveries[deliveryID].Attempts != 1 {
		t.Fatalf("want one attempted delivery, got %d", repo.deliveries[deliveryID].Attempts)
	}
}

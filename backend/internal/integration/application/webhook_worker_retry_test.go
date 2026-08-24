package application

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/intivai/backend/internal/integration/domain"
	shareddomain "github.com/intivai/backend/internal/shared/domain"
	"github.com/rs/zerolog"
)

// failingGetRepo injects a claim-stage failure so the classification of
// GetDeliveryByID/claim errors is observable without a live pool.
type failingGetRepo struct {
	*mockWebhookRepo
	getErr error
}

func (f *failingGetRepo) GetDeliveryByID(ctx context.Context, id uuid.UUID) (*domain.WebhookDelivery, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	return f.mockWebhookRepo.GetDeliveryByID(ctx, id)
}

func newRetryTask(deliveryID, orgID uuid.UUID) *asynq.Task {
	payload, _ := json.Marshal(DeliverWebhookPayload{DeliveryID: deliveryID.String(), OrgID: orgID.String()})
	return asynq.NewTask(TaskDeliverWebhook, payload)
}

func newWorkerFor(repo domain.WebhookRepository) *WebhookWorker {
	return NewWebhookWorker(repo, nil, zerolog.Nop())
}

// Transient claim failure (pool/driver fault) MUST surface as a retryable
// error — returning asynq.SkipRetry here strands the delivery pending
// forever with zero signal (finding D20).
func TestWebhookWorkerTransientClaimFailureRetries(t *testing.T) {
	repo := &failingGetRepo{
		mockWebhookRepo: newMockRepo(),
		getErr:          errors.New("pgx: connection reset by peer"),
	}
	deliveryID, orgID := uuid.New(), uuid.New()

	err := newWorkerFor(repo).handle(txCtx(), newRetryTask(deliveryID, orgID))
	if err == nil || errors.Is(err, asynq.SkipRetry) {
		t.Fatalf("transient claim failure returned %v, want retryable error", err)
	}
	if !errors.Is(err, repo.getErr) {
		t.Fatalf("error %v does not carry the cause for logs", err)
	}
}

// A delivery whose webhook config disappeared can never be delivered again:
// it must be parked failed (terminal), not left pending.
func TestWebhookWorkerConfigRemovedMarksDeliveryFailed(t *testing.T) {
	repo := newMockRepo()
	deliveryID, orgID := uuid.New(), uuid.New()
	repo.deliveries[deliveryID] = &domain.WebhookDelivery{
		Entity:      shareddomain.Entity{ID: deliveryID},
		OrgID:       orgID,
		WebhookID:   uuid.New(), // no matching config
		Event:       domain.EventInterviewCompleted,
		Payload:     []byte(`{}`),
		FinalStatus: "pending",
	}

	err := newWorkerFor(repo).handle(txCtx(), newRetryTask(deliveryID, orgID))
	if !errors.Is(err, asynq.SkipRetry) {
		t.Fatalf("config-missing is permanent, got %v", err)
	}
	got := repo.deliveries[deliveryID]
	if got.FinalStatus != "failed" {
		t.Fatalf("final_status = %q, want failed (delivery must not sit pending forever)", got.FinalStatus)
	}
	if got.ResponseBody == "" {
		t.Fatal("expected a category reason persisted on the delivery")
	}
}

// Wrong-org deliveries must NOT be mutated (another tenant's row) but still
// classify permanent.
func TestWebhookWorkerWrongOrgSkipsWithoutMutation(t *testing.T) {
	repo := &wrongOrgRepo{mockWebhookRepo: newMockRepo()}
	deliveryID, orgID := uuid.New(), uuid.New()
	repo.delivery = &domain.WebhookDelivery{
		Entity:      shareddomain.Entity{ID: deliveryID},
		OrgID:       uuid.New(), // different org
		WebhookID:   uuid.New(),
		Event:       domain.EventInterviewCompleted,
		Payload:     []byte(`{}`),
		FinalStatus: "pending",
	}

	err := newWorkerFor(repo).handle(txCtx(), newRetryTask(deliveryID, orgID))
	if !errors.Is(err, asynq.SkipRetry) {
		t.Fatalf("wrong-org is permanent, got %v", err)
	}
	if repo.delivery.FinalStatus != "pending" {
		t.Fatalf("cross-org row mutated: final_status = %q", repo.delivery.FinalStatus)
	}
}

// wrongOrgRepo simulates the RLS/tenant check failing inside the claim tx by
// returning domain.ErrNotFound after loading a foreign-org delivery.
type wrongOrgRepo struct {
	*mockWebhookRepo
	delivery *domain.WebhookDelivery
}

func (w *wrongOrgRepo) GetDeliveryByID(ctx context.Context, id uuid.UUID) (*domain.WebhookDelivery, error) {
	return w.delivery, nil
}

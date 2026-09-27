package application_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	intapp "github.com/intivai/backend/internal/integration/application"
	intdomain "github.com/intivai/backend/internal/integration/domain"
	notifdomain "github.com/intivai/backend/internal/notification/domain"
	shareddomain "github.com/intivai/backend/internal/shared/domain"
	"github.com/rs/zerolog"
)

type mockWebhookRepoForDispatcher struct {
	configs    map[uuid.UUID][]*intdomain.WebhookConfig
	deliveries []*intdomain.WebhookDelivery
	failCreate bool
}

func (m *mockWebhookRepoForDispatcher) CreateConfig(_ context.Context, _ *intdomain.WebhookConfig) error {
	return nil
}
func (m *mockWebhookRepoForDispatcher) GetConfigByID(_ context.Context, _ uuid.UUID) (*intdomain.WebhookConfig, error) {
	return nil, nil
}
func (m *mockWebhookRepoForDispatcher) ListConfigsByOrg(_ context.Context, _ uuid.UUID) ([]*intdomain.WebhookConfig, error) {
	return nil, nil
}
func (m *mockWebhookRepoForDispatcher) UpdateConfig(_ context.Context, _ *intdomain.WebhookConfig) error {
	return nil
}
func (m *mockWebhookRepoForDispatcher) DeleteConfig(_ context.Context, _ uuid.UUID) error {
	return nil
}
func (m *mockWebhookRepoForDispatcher) CreateDelivery(_ context.Context, d *intdomain.WebhookDelivery) error {
	if m.failCreate {
		return errors.New("db error")
	}
	m.deliveries = append(m.deliveries, d)
	return nil
}
func (m *mockWebhookRepoForDispatcher) GetDeliveryByID(_ context.Context, _ uuid.UUID) (*intdomain.WebhookDelivery, error) {
	return nil, nil
}
func (m *mockWebhookRepoForDispatcher) IncrementDeliveryAttempt(_ context.Context, _ uuid.UUID) (int, bool, error) {
	return 0, false, nil
}
func (m *mockWebhookRepoForDispatcher) ListDeliveriesByWebhook(_ context.Context, _ uuid.UUID, _ int) ([]*intdomain.WebhookDelivery, error) {
	return nil, nil
}
func (m *mockWebhookRepoForDispatcher) UpdateDelivery(_ context.Context, _ *intdomain.WebhookDelivery) error {
	return nil
}
func (m *mockWebhookRepoForDispatcher) UpdateDeliveryIfNotDelivered(_ context.Context, _ *intdomain.WebhookDelivery) (bool, error) {
	return false, nil
}
func (m *mockWebhookRepoForDispatcher) ListConfigsByEvent(_ context.Context, orgID uuid.UUID, _ intdomain.WebhookEvent) ([]*intdomain.WebhookConfig, error) {
	return m.configs[orgID], nil
}

type mockNotificationCreator struct {
	created []notifdomain.EventType
}

func (m *mockNotificationCreator) Create(_ context.Context, _ uuid.UUID, eventType notifdomain.EventType, _, _ string, _ string) error {
	m.created = append(m.created, eventType)
	return nil
}

type mockQueueClient struct {
	tasks       []string
	payloads    [][]byte
	rawPayloads []any
	failOnTask  string
}

func (m *mockQueueClient) Enqueue(_ context.Context, jobType string, payload any, _ ...asynq.Option) (*asynq.TaskInfo, error) {
	if m.failOnTask == jobType {
		return nil, errors.New("queue error")
	}
	m.tasks = append(m.tasks, jobType)
	m.rawPayloads = append(m.rawPayloads, payload)
	if b, ok := payload.([]byte); ok {
		m.payloads = append(m.payloads, b)
	} else if b, err := json.Marshal(payload); err == nil {
		m.payloads = append(m.payloads, b)
	}
	return nil, nil
}

func TestEventDispatcher_Dispatch_InterviewCompleted(t *testing.T) {
	orgID := uuid.New()
	whID1 := uuid.New()
	whID2 := uuid.New()

	whRepo := &mockWebhookRepoForDispatcher{
		configs: map[uuid.UUID][]*intdomain.WebhookConfig{
			orgID: {
				{Entity: shareddomain.Entity{ID: whID1}, OrgID: orgID, URL: "https://example.com/wh1", Active: true},
				{Entity: shareddomain.Entity{ID: whID2}, OrgID: orgID, URL: "https://example.com/wh2", Active: true},
			},
		},
	}
	notifs := &mockNotificationCreator{}
	q := &mockQueueClient{}
	logger := zerolog.Nop()

	dispatcher := intapp.NewEventDispatcher(nil, whRepo, notifs, q, logger)

	payload := []byte(`{"interview_id":"123","score":95.5}`)
	err := dispatcher.Dispatch(context.Background(), orgID, intdomain.EventInterviewCompleted, payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(notifs.created) != 1 || notifs.created[0] != notifdomain.EventInterviewCompleted {
		t.Errorf("expected 1 notification with EventInterviewCompleted, got %v", notifs.created)
	}

	if len(whRepo.deliveries) != 2 {
		t.Fatalf("expected 2 deliveries created, got %d", len(whRepo.deliveries))
	}

	for _, d := range whRepo.deliveries {
		if d.OrgID != orgID {
			t.Errorf("expected OrgID %s, got %s", orgID, d.OrgID)
		}
		if d.Event != intdomain.EventInterviewCompleted {
			t.Errorf("expected Event interview.completed, got %s", d.Event)
		}
		if d.FinalStatus != "pending" {
			t.Errorf("expected FinalStatus pending, got %s", d.FinalStatus)
		}
	}

	if len(q.tasks) != 2 {
		t.Fatalf("expected 2 tasks enqueued, got %d", len(q.tasks))
	}
	for i, task := range q.tasks {
		if task != intapp.TaskDeliverWebhook {
			t.Errorf("expected task %s, got %s", intapp.TaskDeliverWebhook, task)
		}
		var pl intapp.DeliverWebhookPayload
		if err := json.Unmarshal(q.payloads[i], &pl); err != nil {
			t.Errorf("failed to unmarshal payload: %v", err)
		}
		if pl.OrgID != orgID.String() {
			t.Errorf("expected org_id %s, got %s", orgID, pl.OrgID)
		}
	}
}

func TestEventDispatcher_Dispatch_NoConfigs_StillNotifies(t *testing.T) {
	orgID := uuid.New()
	whRepo := &mockWebhookRepoForDispatcher{
		configs: map[uuid.UUID][]*intdomain.WebhookConfig{},
	}
	notifs := &mockNotificationCreator{}
	q := &mockQueueClient{}

	dispatcher := intapp.NewEventDispatcher(nil, whRepo, notifs, q, zerolog.Nop())

	err := dispatcher.Dispatch(context.Background(), orgID, intdomain.EventInterviewCompleted, []byte(`{}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(notifs.created) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(notifs.created))
	}
	if len(whRepo.deliveries) != 0 {
		t.Fatalf("expected 0 deliveries, got %d", len(whRepo.deliveries))
	}
	if len(q.tasks) != 0 {
		t.Fatalf("expected 0 tasks, got %d", len(q.tasks))
	}
}

func TestEventDispatcher_Dispatch_QueueFailure_ReturnsError(t *testing.T) {
	orgID := uuid.New()
	whRepo := &mockWebhookRepoForDispatcher{
		configs: map[uuid.UUID][]*intdomain.WebhookConfig{
			orgID: {
				{Entity: shareddomain.Entity{ID: uuid.New()}, OrgID: orgID, URL: "https://example.com/wh", Active: true},
			},
		},
	}
	notifs := &mockNotificationCreator{}
	q := &mockQueueClient{failOnTask: intapp.TaskDeliverWebhook}

	dispatcher := intapp.NewEventDispatcher(nil, whRepo, notifs, q, zerolog.Nop())

	err := dispatcher.Dispatch(context.Background(), orgID, intdomain.EventInterviewCompleted, []byte(`{}`))
	if err == nil {
		t.Fatal("expected error on queue failure, got nil")
	}
}

func TestEventDispatcher_Dispatch_CreateDeliveryFailure_ReturnsError(t *testing.T) {
	orgID := uuid.New()
	whRepo := &mockWebhookRepoForDispatcher{
		configs: map[uuid.UUID][]*intdomain.WebhookConfig{
			orgID: {
				{Entity: shareddomain.Entity{ID: uuid.New()}, OrgID: orgID, URL: "https://example.com/wh", Active: true},
			},
		},
		failCreate: true,
	}
	notifs := &mockNotificationCreator{}
	q := &mockQueueClient{}

	dispatcher := intapp.NewEventDispatcher(nil, whRepo, notifs, q, zerolog.Nop())

	err := dispatcher.Dispatch(context.Background(), orgID, intdomain.EventInterviewCompleted, []byte(`{}`))
	if err == nil {
		t.Fatal("expected error on CreateDelivery failure, got nil")
	}
}

package application_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/hibiken/asynq"
	memapp "github.com/intivai/backend/internal/memory/application"
	memdomain "github.com/intivai/backend/internal/memory/domain"
)

type mockMemoryBank struct {
	rememberFn func(ctx context.Context, entityType, summary string, importance float64) error
	recallFn   func(ctx context.Context, query string, budget string) ([]memdomain.GroundingHit, error)
}

func (m *mockMemoryBank) Remember(ctx context.Context, entityType, summary string, importance float64) error {
	if m.rememberFn != nil {
		return m.rememberFn(ctx, entityType, summary, importance)
	}
	return nil
}

func (m *mockMemoryBank) Recall(ctx context.Context, query string, budget string) ([]memdomain.GroundingHit, error) {
	if m.recallFn != nil {
		return m.recallFn(ctx, query, budget)
	}
	return nil, nil
}

type mockBankFactory struct {
	bank memdomain.GroundingBank
}

func (f *mockBankFactory) ForBank(_ string) memdomain.GroundingBank {
	return f.bank
}

func TestSyncWorker_HandleSync_Success(t *testing.T) {
	var capturedEntity, capturedSummary string
	var capturedImportance float64

	mockBank := &mockMemoryBank{
		rememberFn: func(_ context.Context, entityType, summary string, importance float64) error {
			capturedEntity = entityType
			capturedSummary = summary
			capturedImportance = importance
			return nil
		},
	}
	factory := &mockBankFactory{bank: mockBank}
	worker := memapp.NewSyncWorker(factory)

	payload, _ := json.Marshal(memapp.SyncPayload{
		OrgID:      "org-123",
		EntityType: "candidate_profile",
		Summary:    "Senior Go engineer with distributed systems exp",
		Importance: 0.85,
	})
	task := asynq.NewTask(memapp.TaskSyncMnemosyne, payload)

	mux := asynq.NewServeMux()
	worker.Register(mux)

	handler := worker.HandleSync
	err := handler(context.Background(), task)
	if err != nil {
		t.Fatalf("expected nil err, got: %v", err)
	}

	if capturedEntity != "candidate_profile" || capturedSummary != "Senior Go engineer with distributed systems exp" || capturedImportance != 0.85 {
		t.Errorf("unexpected captured fields: %s, %s, %f", capturedEntity, capturedSummary, capturedImportance)
	}
}

func TestSyncWorker_HandleSync_MissingRequiredFields_SkipsRetry(t *testing.T) {
	factory := &mockBankFactory{bank: &mockMemoryBank{}}
	worker := memapp.NewSyncWorker(factory)

	tests := []struct {
		name    string
		payload memapp.SyncPayload
	}{
		{"missing orgID", memapp.SyncPayload{EntityType: "cand", Summary: "sum"}},
		{"missing entityType", memapp.SyncPayload{OrgID: "org-1", Summary: "sum"}},
		{"missing summary", memapp.SyncPayload{OrgID: "org-1", EntityType: "cand"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b, _ := json.Marshal(tc.payload)
			task := asynq.NewTask(memapp.TaskSyncMnemosyne, b)
			err := worker.HandleSync(context.Background(), task)
			if !errors.Is(err, asynq.SkipRetry) {
				t.Fatalf("expected asynq.SkipRetry, got %v", err)
			}
		})
	}
}

func TestSyncWorker_HandleSync_MalformedPayload_ReturnsError(t *testing.T) {
	factory := &mockBankFactory{bank: &mockMemoryBank{}}
	worker := memapp.NewSyncWorker(factory)

	task := asynq.NewTask(memapp.TaskSyncMnemosyne, []byte("invalid-json"))
	err := worker.HandleSync(context.Background(), task)
	if !errors.Is(err, asynq.SkipRetry) {
		t.Fatalf("expected asynq.SkipRetry on malformed payload, got %v", err)
	}
}

func TestSyncWorker_HandleSync_BankError_Propagates(t *testing.T) {
	expectedErr := errors.New("db connection failure")
	mockBank := &mockMemoryBank{
		rememberFn: func(_ context.Context, _, _ string, _ float64) error {
			return expectedErr
		},
	}
	factory := &mockBankFactory{bank: mockBank}
	worker := memapp.NewSyncWorker(factory)

	payload, _ := json.Marshal(memapp.SyncPayload{
		OrgID:      "org-123",
		EntityType: "candidate_profile",
		Summary:    "Senior Go engineer",
		Importance: 0.8,
	})
	task := asynq.NewTask(memapp.TaskSyncMnemosyne, payload)

	err := worker.HandleSync(context.Background(), task)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected %v, got %v", expectedErr, err)
	}
}

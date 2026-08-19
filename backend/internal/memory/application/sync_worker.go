package application

import (
	"context"
	"encoding/json"

	"github.com/hibiken/asynq"
	"github.com/intivai/backend/internal/memory/domain"
)

// Job types — workers consumed by the asynq server registered in cmd/server.
const (
	TaskSyncMnemosyne = "sync_mnemosyne"
)

// SyncPayload is the task payload for TaskSyncMnemosyne.
type SyncPayload struct {
	OrgID      string  `json:"org_id"`
	EntityType string  `json:"entity_type"`
	Summary    string  `json:"summary"`
	Importance float64 `json:"importance"`
}

// SyncWorker writes per-tenant Mnemosyne bank entries. Producers (CV extract,
// context index) enqueue TaskSyncMnemosyne; the handler below is the only sink.
type SyncWorker struct {
	factory domain.BankFactory
}

func NewSyncWorker(factory domain.BankFactory) *SyncWorker {
	return &SyncWorker{factory: factory}
}

// Register adds the sync_mnemosyne handler to the shared asynq mux.
func (s *SyncWorker) Register(mux *asynq.ServeMux) {
	mux.HandleFunc(TaskSyncMnemosyne, s.handleSync)
}

func (s *SyncWorker) handleSync(ctx context.Context, t *asynq.Task) error {
	var p SyncPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return err
	}
	if p.OrgID == "" || p.EntityType == "" || p.Summary == "" {
		return asynq.SkipRetry
	}
	return s.factory.ForBank(p.OrgID).Remember(ctx, p.EntityType, p.Summary, p.Importance)
}

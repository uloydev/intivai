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

// SyncWorker skeleton (Phase 1): per-tenant Mnemosyne bank writes.
// M2 wires real producers (ParseCV/ExtractCV/IndexContext) into these handlers.
type SyncWorker struct {
	factory domain.BankFactory
}

func NewSyncWorker(factory domain.BankFactory) *SyncWorker {
	return &SyncWorker{factory: factory}
}

// SyncCandidate remembers a candidate profile into the tenant bank.
func (s *SyncWorker) SyncCandidate(ctx context.Context, orgID, candidateID, summary string) error {
	return s.factory.ForBank(orgID).Remember(ctx, "candidate_profile", summary, 0.9)
}

// SyncContext indexes company context into the tenant bank.
func (s *SyncWorker) SyncContext(ctx context.Context, orgID, contextType, summary string) error {
	return s.factory.ForBank(orgID).Remember(ctx, "company_context_"+contextType, summary, 0.8)
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

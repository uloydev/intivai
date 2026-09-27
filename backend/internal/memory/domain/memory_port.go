package domain

import (
	"context"
)

// GroundingHit represents a recalled semantic or keyword memory entry.
type GroundingHit struct {
	ID      string
	Content string
	Score   float64
}

// MemoryHit is an alias for GroundingHit.
type MemoryHit = GroundingHit

// GroundingBank is the focused port for semantic grounding and contextual recall.
// Speculative reflection, graph query, stats, and delete operations are removed (deletion test).
type GroundingBank interface {
	Remember(ctx context.Context, entityType, summary string, importance float64) error
	Recall(ctx context.Context, query string, budget string) ([]GroundingHit, error)
}

// MemoryBank alias for GroundingBank for backward compatibility.
type MemoryBank = GroundingBank

// BankFactory creates a per-tenant grounding bank ensuring isolation.
type BankFactory interface {
	ForBank(orgID string) GroundingBank
}

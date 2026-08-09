package domain

import (
	"context"
)

// MemoryBank — the Go port for the semantic memory layer.
// One port, two adapter options (research docs §4.3):
//
//	Option A: Mnemosyne MCP (stdio/HTTP) — features limited to the plugin
//	Option B: Native Go (recommended) — SQLite + fastembed + BM25 + LLM reflect
//
// The decision can be deferred: adapters swap without touching use cases.
type MemoryBank interface {
	Remember(ctx context.Context, entityType, summary string, importance float64) error
	Recall(ctx context.Context, query string, budget string) ([]MemoryHit, error)
	Reflect(ctx context.Context, question string) (string, error)
	QueryGraph(ctx context.Context, entityType, filter string) ([]MemoryHit, error)
	Forget(ctx context.Context, memoryID string) error
	Stats(ctx context.Context) (MemoryStats, error)
}

type MemoryHit struct {
	ID      string
	Content string
	Score   float64
}

type MemoryStats struct {
	Banks     int
	Memories  int
	Embedding string
}

// BankFactory creates a per-tenant bank — isolation at file/database level.
type BankFactory interface {
	ForBank(orgID string) MemoryBank
}

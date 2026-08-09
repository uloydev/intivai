package native

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite" // pure-Go SQLite driver (no CGO, single binary)

	memdomain "github.com/intivai/backend/internal/memory/domain"
)

// NativeMemory — Option B (recommended): SQLite bank per tenant.
// Embeddings via fastembed (bge-small, 384 dims) land in M2; the storage
// schema and Recall/Reflect/QueryGraph surface are ready now.
type NativeMemory struct {
	orgID string
	path  string
	db    *sql.DB
}

// NativeFactory opens one SQLite bank file per tenant under dataDir/banks/<org_id>/mnemosyne.db
type NativeFactory struct {
	dataDir string
}

func NewNativeFactory(dataDir string) *NativeFactory {
	return &NativeFactory{dataDir: dataDir}
}

func (f *NativeFactory) ForBank(orgID string) memdomain.MemoryBank {
	return &NativeMemory{orgID: orgID, path: filepath.Join(f.dataDir, "banks", orgID, "mnemosyne.db")}
}

func (m *NativeMemory) open() error {
	if m.db != nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(m.path), 0o755); err != nil {
		return fmt.Errorf("create bank dir: %w", err)
	}
	db, err := sql.Open("sqlite", m.path)
	if err != nil {
		return fmt.Errorf("open sqlite: %w", err)
	}
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return fmt.Errorf("init sqlite schema: %w", err)
	}
	m.db = db
	return nil
}

const schema = `
CREATE TABLE IF NOT EXISTS memories (
    id          TEXT PRIMARY KEY,
    entity_type TEXT NOT NULL,
    content     TEXT NOT NULL,
    importance  REAL NOT NULL DEFAULT 0.5,
    embedding   BLOB,          -- float32 array, fastembed (384 dims) — M2
    filter      TEXT,          -- structured filter for QueryGraph, e.g. "passed_screening=true"
    created_at  TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_memories_entity ON memories(entity_type);
CREATE INDEX IF NOT EXISTS idx_memories_filter  ON memories(filter);
`

func (m *NativeMemory) Remember(ctx context.Context, entityType, summary string, importance float64) error {
	if err := m.open(); err != nil {
		return err
	}
	_, err := m.db.ExecContext(ctx,
		`INSERT INTO memories (id, entity_type, content, importance) VALUES (?, ?, ?, ?)`,
		newID(), entityType, summary, importance)
	return err
}

func (m *NativeMemory) Recall(ctx context.Context, query string, budget string) ([]memdomain.MemoryHit, error) {
	if err := m.open(); err != nil {
		return nil, err
	}
	// M2: BM25 + embedding similarity. M1: keyword LIKE recall so the port works end-to-end.
	rows, err := m.db.QueryContext(ctx,
		`SELECT id, content, importance FROM memories WHERE content LIKE '%' || ? || '%' ORDER BY importance DESC LIMIT 20`, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanHits(rows)
}

func (m *NativeMemory) QueryGraph(ctx context.Context, entityType, filter string) ([]memdomain.MemoryHit, error) {
	if err := m.open(); err != nil {
		return nil, err
	}
	rows, err := m.db.QueryContext(ctx,
		`SELECT id, content, importance FROM memories WHERE entity_type = ? AND (filter = ? OR filter IS NULL) ORDER BY importance DESC LIMIT 50`,
		entityType, filter)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanHits(rows)
}

func (m *NativeMemory) Reflect(ctx context.Context, question string) (string, error) {
	// M2: aggregate recall + LLM synthesis. Needs the LLM provider wired in.
	return "", errors.New("reflect not implemented until LLM provider is wired (M2)")
}

func (m *NativeMemory) Forget(ctx context.Context, memoryID string) error {
	if err := m.open(); err != nil {
		return err
	}
	_, err := m.db.ExecContext(ctx, `DELETE FROM memories WHERE id = ?`, memoryID)
	return err
}

func (m *NativeMemory) Stats(ctx context.Context) (memdomain.MemoryStats, error) {
	if err := m.open(); err != nil {
		return memdomain.MemoryStats{}, err
	}
	var n int
	if err := m.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM memories`).Scan(&n); err != nil {
		return memdomain.MemoryStats{}, err
	}
	return memdomain.MemoryStats{Banks: 1, Memories: n, Embedding: "bge-small-en-v1.5 (384d, M2)"}, nil
}

func (m *NativeMemory) Close() error {
	if m.db != nil {
		return m.db.Close()
	}
	return nil
}

func scanHits(rows *sql.Rows) ([]memdomain.MemoryHit, error) {
	hits := []memdomain.MemoryHit{}
	for rows.Next() {
		var h memdomain.MemoryHit
		var imp float64
		if err := rows.Scan(&h.ID, &h.Content, &imp); err != nil {
			return nil, err
		}
		h.Score = imp
		hits = append(hits, h)
	}
	return hits, rows.Err()
}

func newID() string {
	b := make([]byte, 16)
	_, _ = randRead(b)
	return fmt.Sprintf("%x", b)
}

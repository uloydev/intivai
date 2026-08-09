package postgres

import (
	"context"
	"database/sql"
	"errors"

	memdomain "github.com/intivai/backend/internal/memory/domain"
	"github.com/intivai/backend/pkg/db"
	"gorm.io/gorm"
)

// PostgresBank — MemoryBank backed by the pgvector table mnemosyne_memories.
// 1 bank = rows filtered by org_id; tenant isolation via RLS (defense in
// depth: every op runs in a transaction with app.org_id set).
// Swap target for the SQLite native adapter in production.
type PostgresBank struct {
	pool  *gorm.DB
	orgID string
}

// PostgresFactory creates one bank handle per tenant (org_id partition).
type PostgresFactory struct {
	pool *gorm.DB
}

func NewPostgresFactory(pool *gorm.DB) *PostgresFactory {
	return &PostgresFactory{pool: pool}
}

func (f *PostgresFactory) ForBank(orgID string) memdomain.MemoryBank {
	return &PostgresBank{pool: f.pool, orgID: orgID}
}

// withTenant runs fn inside a transaction with app.org_id set so the RLS
// policy resolves. Equivalent to the tenant-tx middleware used by the API.
func (b *PostgresBank) withTenant(ctx context.Context, fn func(tx *gorm.DB) error) error {
	tx := b.pool.WithContext(ctx).Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer tx.Rollback()

	if err := db.SetTenant(ctx, tx, b.orgID); err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit().Error
}

func (b *PostgresBank) Remember(ctx context.Context, entityType, summary string, importance float64) error {
	return b.withTenant(ctx, func(tx *gorm.DB) error {
		return tx.Exec(
			`INSERT INTO mnemosyne_memories (org_id, entity_type, content, importance)
			 VALUES ($1, $2, $3, $4)`,
			b.orgID, entityType, summary, importance).Error
	})
}

func (b *PostgresBank) Recall(ctx context.Context, query, budget string) ([]memdomain.MemoryHit, error) {
	return b.query(ctx,
		`SELECT id, content, importance FROM mnemosyne_memories
		 WHERE content LIKE '%' || $1 || '%' ORDER BY importance DESC LIMIT 20`, query)
}

func (b *PostgresBank) QueryGraph(ctx context.Context, entityType, filter string) ([]memdomain.MemoryHit, error) {
	return b.query(ctx,
		`SELECT id, content, importance FROM mnemosyne_memories
		 WHERE entity_type = $1 AND (filter = $2 OR filter IS NULL) ORDER BY importance DESC LIMIT 50`,
		entityType, filter)
}

func (b *PostgresBank) query(ctx context.Context, sql string, args ...any) ([]memdomain.MemoryHit, error) {
	var hits []memdomain.MemoryHit
	err := b.withTenant(ctx, func(tx *gorm.DB) error {
		rows, err := tx.Raw(sql, args...).Rows()
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var h memdomain.MemoryHit
			var imp float64
			if err := rows.Scan(&h.ID, &h.Content, &imp); err != nil {
				return err
			}
			h.Score = imp
			hits = append(hits, h)
		}
		return rows.Err()
	})
	return hits, err
}

func (b *PostgresBank) Reflect(ctx context.Context, question string) (string, error) {
	// M2: aggregate recall + LLM synthesis. Needs the LLM provider wired in.
	return "", errors.New("reflect not implemented until LLM provider is wired (M2)")
}

func (b *PostgresBank) Forget(ctx context.Context, memoryID string) error {
	return b.withTenant(ctx, func(tx *gorm.DB) error {
		return tx.Exec(`DELETE FROM mnemosyne_memories WHERE id = $1`, memoryID).Error
	})
}

func (b *PostgresBank) Stats(ctx context.Context) (memdomain.MemoryStats, error) {
	stats := memdomain.MemoryStats{
		Banks:     1,
		Embedding: "bge-small-en-v1.5 (384d, M2)",
	}
	err := b.withTenant(ctx, func(tx *gorm.DB) error {
		return tx.Raw(`SELECT COUNT(*) FROM mnemosyne_memories`).Row().Scan(&stats.Memories)
	})
	if errors.Is(err, sql.ErrNoRows) {
		return stats, nil
	}
	return stats, err
}

func (b *PostgresBank) Close() error { return nil }

var _ memdomain.MemoryBank = (*PostgresBank)(nil)
var _ memdomain.BankFactory = (*PostgresFactory)(nil)

package llm

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// TokenLedger tracks LLM token usage per organization to prevent unbounded spend.
type TokenLedger interface {
	// CheckAndRecord verifies if the org has enough budget and records the usage.
	// Returns ErrRateLimited if the daily budget is exceeded.
	CheckAndRecord(ctx context.Context, orgID string, tokens int) error
}

type RedisTokenLedger struct {
	rdb      *redis.Client
	dailyCap int
}

func NewRedisTokenLedger(rdb *redis.Client, dailyCap int) *RedisTokenLedger {
	return &RedisTokenLedger{rdb: rdb, dailyCap: dailyCap}
}

func (l *RedisTokenLedger) CheckAndRecord(ctx context.Context, orgID string, tokens int) error {
	if l.dailyCap <= 0 {
		return nil // No cap
	}

	key := fmt.Sprintf("llm:usage:org:%s:%s", orgID, time.Now().Format("2006-01-02"))

	val, err := l.rdb.IncrBy(ctx, key, int64(tokens)).Result()
	if err != nil {
		return err
	}

	if val == int64(tokens) {
		// First time today, set expiration for 24h + some buffer
		_ = l.rdb.Expire(ctx, key, 48*time.Hour)
	}

	if val > int64(l.dailyCap) {
		return ErrRateLimited
	}

	return nil
}

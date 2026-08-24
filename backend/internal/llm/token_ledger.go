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

const recordUsageScript = `
local value = redis.call("INCRBY", KEYS[1], ARGV[1])
if value == tonumber(ARGV[1]) then
  redis.call("EXPIRE", KEYS[1], ARGV[2])
end
return value
`

func NewRedisTokenLedger(rdb *redis.Client, dailyCap int) *RedisTokenLedger {
	return &RedisTokenLedger{rdb: rdb, dailyCap: dailyCap}
}

func (l *RedisTokenLedger) CheckAndRecord(ctx context.Context, orgID string, tokens int) error {
	if l.dailyCap <= 0 {
		return nil // No cap
	}

	key := fmt.Sprintf("llm:usage:org:%s:%s", orgID, time.Now().Format("2006-01-02"))

	val, err := l.rdb.Eval(ctx, recordUsageScript, []string{key}, tokens, int64((48 * time.Hour).Seconds())).Int64()
	if err != nil {
		return err
	}

	if val > int64(l.dailyCap) {
		return ErrRateLimited
	}

	return nil
}

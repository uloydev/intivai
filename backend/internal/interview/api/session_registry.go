package api

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// SessionRegistry enforces the single-active-connection constraint per interview.
// When an interview already has an active WebSocket session, a second socket
// is rejected.
type SessionRegistry interface {
	TryAcquire(ctx context.Context, key, sessionID string) (bool, error)
	Release(ctx context.Context, key, sessionID string) error
	// Touch extends the lock's lifetime for the CURRENT holder — called from
	// the connection heartbeat so a long-lived active connection never loses
	// its lock to TTL expiry.
	Touch(ctx context.Context, key, sessionID string) error
}

// MemorySessionRegistry is a thread-safe in-memory session registry.
type MemorySessionRegistry struct {
	mu     sync.Mutex
	active map[string]string
}

func NewMemorySessionRegistry() *MemorySessionRegistry {
	return &MemorySessionRegistry{active: make(map[string]string)}
}

// TryAcquire claims the key; returns false when already held by another session.
func (r *MemorySessionRegistry) TryAcquire(_ context.Context, key, sessionID string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if holder, ok := r.active[key]; ok {
		if sessionID != "" && holder == sessionID {
			return true, nil
		}
		return false, nil
	}
	r.active[key] = sessionID
	return true, nil
}

// Release frees the key (idempotent; only deletes if matching sessionID or empty).
func (r *MemorySessionRegistry) Release(_ context.Context, key, sessionID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if holder, ok := r.active[key]; ok {
		if sessionID == "" || holder == sessionID {
			delete(r.active, key)
		}
	}
	return nil
}

// Touch is a no-op for the in-memory registry — the lock lives as long as
// the registry does.
func (r *MemorySessionRegistry) Touch(_ context.Context, key, sessionID string) error {
	return nil
}

const (
	// activeSessionPlaceholder marks the Redis key holder when no concrete
	// session ID is supplied (pre-session clients still occupy the lock).
	activeSessionPlaceholder = "active"

	redisSessionPrefix = "intivai:session:"
	// acquireLua: claim the key, OR refresh the TTL when the SAME session
	// re-acquires (reconnect / resume) — a long-lived connection must not
	// let its lock lapse and admit a second socket mid-interview.
	acquireLuaScript = `
local holder = redis.call("get", KEYS[1])
if holder == false then
    redis.call("set", KEYS[1], ARGV[1], "PX", ARGV[2])
    return 1
elseif holder == ARGV[1] then
    redis.call("pexpire", KEYS[1], ARGV[2])
    return 1
end
return 0`
	touchLuaScript = `
if redis.call("get", KEYS[1]) == ARGV[1] then
    return redis.call("pexpire", KEYS[1], ARGV[2])
end
return 0`
	releaseLuaScript = `
if redis.call("get", KEYS[1]) == ARGV[1] then
    return redis.call("del", KEYS[1])
else
    return 0
end`
)

// RedisSessionRegistry manages distributed interview session locks in Redis.
type RedisSessionRegistry struct {
	client redis.UniversalClient
	ttl    time.Duration
}

func NewRedisSessionRegistry(client redis.UniversalClient, ttl time.Duration) *RedisSessionRegistry {
	if ttl <= 0 {
		ttl = 30 * time.Minute
	}
	return &RedisSessionRegistry{client: client, ttl: ttl}
}

func (r *RedisSessionRegistry) TryAcquire(ctx context.Context, key, sessionID string) (bool, error) {
	if r.client == nil {
		return true, nil
	}
	redisKey := redisSessionPrefix + key
	val := sessionID
	if val == "" {
		val = activeSessionPlaceholder
	}
	res, err := r.client.Eval(ctx, acquireLuaScript, []string{redisKey}, val, r.ttl.Milliseconds()).Result()
	if err != nil {
		return false, err
	}
	ok, _ := res.(int64)
	return ok == 1, nil
}

// Touch extends the TTL when sessionID still holds the key.
func (r *RedisSessionRegistry) Touch(ctx context.Context, key, sessionID string) error {
	if r.client == nil || sessionID == "" {
		return nil
	}
	redisKey := redisSessionPrefix + key
	_, err := r.client.Eval(ctx, touchLuaScript, []string{redisKey}, sessionID, r.ttl.Milliseconds()).Result()
	if errors.Is(err, redis.Nil) {
		return nil
	}
	return err
}

func (r *RedisSessionRegistry) Release(ctx context.Context, key, sessionID string) error {
	if r.client == nil {
		return nil
	}
	redisKey := redisSessionPrefix + key
	val := sessionID
	if val == "" {
		_, err := r.client.Del(ctx, redisKey).Result()
		return err
	}
	_, err := r.client.Eval(ctx, releaseLuaScript, []string{redisKey}, val).Result()
	if errors.Is(err, redis.Nil) {
		return nil
	}
	return err
}

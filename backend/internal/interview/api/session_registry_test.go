package api

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func TestMemorySessionRegistry(t *testing.T) {
	ctx := context.Background()
	r := NewMemorySessionRegistry()

	ok, err := r.TryAcquire(ctx, "iv-1", "sess-1")
	if err != nil || !ok {
		t.Fatalf("first acquire failed: ok=%v, err=%v", ok, err)
	}

	// Re-acquire with same sessionID should succeed
	ok, err = r.TryAcquire(ctx, "iv-1", "sess-1")
	if err != nil || !ok {
		t.Fatalf("same session re-acquire failed: ok=%v, err=%v", ok, err)
	}

	// Acquire with different sessionID should fail
	ok, err = r.TryAcquire(ctx, "iv-1", "sess-2")
	if err != nil || ok {
		t.Fatalf("different session acquire succeeded: ok=%v, err=%v", ok, err)
	}

	// Different key should succeed
	ok, err = r.TryAcquire(ctx, "iv-2", "sess-2")
	if err != nil || !ok {
		t.Fatalf("different key acquire failed: ok=%v, err=%v", ok, err)
	}

	// Release with wrong sessionID does not release
	if err := r.Release(ctx, "iv-1", "sess-wrong"); err != nil {
		t.Fatal(err)
	}
	ok, err = r.TryAcquire(ctx, "iv-1", "sess-3")
	if err != nil || ok {
		t.Fatal("key should still be locked by sess-1")
	}

	// Release with matching sessionID
	if err := r.Release(ctx, "iv-1", "sess-1"); err != nil {
		t.Fatal(err)
	}
	ok, err = r.TryAcquire(ctx, "iv-1", "sess-3")
	if err != nil || !ok {
		t.Fatalf("acquire after release failed: ok=%v, err=%v", ok, err)
	}

	// Idempotent release
	if err := r.Release(ctx, "iv-1", "sess-3"); err != nil {
		t.Fatal(err)
	}
	if err := r.Release(ctx, "iv-1", "sess-3"); err != nil {
		t.Fatal(err)
	}
}

func TestRedisSessionRegistry(t *testing.T) {
	redisAddr := os.Getenv("TEST_REDIS_ADDR")
	if redisAddr == "" {
		t.Skip("TEST_REDIS_ADDR not set, skipping RedisSessionRegistry test")
	}

	ctx := context.Background()
	client := redis.NewClient(&redis.Options{Addr: redisAddr})
	defer func() { _ = client.Close() }()

	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("redis ping failed: %v", err)
	}

	r := NewRedisSessionRegistry(client, 10*time.Second)
	testKey := "test-iv-reg-" + time.Now().Format("150405.000000")

	// Acquire
	ok, err := r.TryAcquire(ctx, testKey, "sess-a")
	if err != nil || !ok {
		t.Fatalf("first acquire failed: ok=%v, err=%v", ok, err)
	}

	// Acquire same key with different session -> false
	ok, err = r.TryAcquire(ctx, testKey, "sess-b")
	if err != nil || ok {
		t.Fatalf("second acquire should fail: ok=%v, err=%v", ok, err)
	}

	// Release with wrong session -> key still exists
	if err := r.Release(ctx, testKey, "sess-wrong"); err != nil {
		t.Fatal(err)
	}
	ok, err = r.TryAcquire(ctx, testKey, "sess-b")
	if err != nil || ok {
		t.Fatal("key should still be held")
	}

	// Release with correct session -> released
	if err := r.Release(ctx, testKey, "sess-a"); err != nil {
		t.Fatal(err)
	}
	ok, err = r.TryAcquire(ctx, testKey, "sess-b")
	if err != nil || !ok {
		t.Fatalf("acquire after release failed: ok=%v, err=%v", ok, err)
	}

	// Cleanup
	_ = r.Release(ctx, testKey, "sess-b")
}

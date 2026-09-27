package queue

import (
	"testing"

	"github.com/hibiken/asynq"
)

func TestParseRedisConnOpt_Default(t *testing.T) {
	asynqOpt, redisOpt, err := ParseRedisConnOpt(RedisConfig{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if clientOpt, ok := asynqOpt.(asynq.RedisClientOpt); !ok || clientOpt.Addr != "localhost:6379" {
		t.Errorf("asynqOpt = %+v, want Addr localhost:6379", asynqOpt)
	}
	if redisOpt == nil || redisOpt.Addr != "localhost:6379" || redisOpt.Password != "" {
		t.Errorf("redisOpt = %+v, want Addr localhost:6379 empty password", redisOpt)
	}
}

func TestParseRedisConnOpt_AddrAndPassword(t *testing.T) {
	asynqOpt, redisOpt, err := ParseRedisConnOpt(RedisConfig{
		Addr:     "redis.internal:6380",
		Password: "supersecretpassword",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if clientOpt, ok := asynqOpt.(asynq.RedisClientOpt); !ok || clientOpt.Addr != "redis.internal:6380" || clientOpt.Password != "supersecretpassword" {
		t.Errorf("asynqOpt = %+v, want Addr redis.internal:6380 password", asynqOpt)
	}
	if redisOpt == nil || redisOpt.Addr != "redis.internal:6380" || redisOpt.Password != "supersecretpassword" {
		t.Errorf("redisOpt = %+v, want Addr redis.internal:6380 password", redisOpt)
	}
}

func TestParseRedisConnOpt_URL(t *testing.T) {
	asynqOpt, redisOpt, err := ParseRedisConnOpt(RedisConfig{
		URL: "redis://:secretpass@redis.prod:6379/2",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if asynqOpt == nil {
		t.Fatal("asynqOpt is nil")
	}
	if redisOpt == nil || redisOpt.Addr != "redis.prod:6379" || redisOpt.Password != "secretpass" || redisOpt.DB != 2 {
		t.Errorf("redisOpt = %+v, want Addr redis.prod:6379 password secretpass db 2", redisOpt)
	}
}

func TestParseRedisConnOpt_InvalidURL(t *testing.T) {
	_, _, err := ParseRedisConnOpt(RedisConfig{
		URL: "://invalid-url",
	})
	if err == nil {
		t.Error("expected error for invalid URL, got nil")
	}
}

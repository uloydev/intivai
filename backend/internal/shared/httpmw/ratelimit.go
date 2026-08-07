package httpmw

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
)

// RateLimit implements a Redis sliding-window counter per key.
// keyFn builds the bucket key (e.g. "rl:tenant:{org_id}"); nil key = unlimited.
func RateLimit(rdb *redis.Client, limit int, window time.Duration, keyFn func(c *fiber.Ctx) string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		key := ""
		if keyFn != nil {
			key = keyFn(c)
		}
		if key == "" {
			return c.Next()
		}

		ctx, cancel := context.WithTimeout(c.UserContext(), 500*time.Millisecond)
		defer cancel()

		now := float64(time.Now().UnixNano()) / 1e9
		cutoff := now - window.Seconds()
		redisKey := fmt.Sprintf("rl:%s", key)

		pipe := rdb.TxPipeline()
		pipe.ZRemRangeByScore(ctx, redisKey, "0", strconv.FormatFloat(cutoff, 'f', 0, 64))
		pipe.ZAdd(ctx, redisKey, redis.Z{Score: now, Member: fmt.Sprintf("%d", time.Now().UnixNano())})
		countCmd := pipe.ZCard(ctx, redisKey)
		pipe.Expire(ctx, redisKey, window*2)
		if _, err := pipe.Exec(ctx); err != nil {
			return c.Next() // fail open on Redis errors; auth/audit still protected
		}

		if countCmd.Val() > int64(limit) {
			retry := int64(window.Seconds()) - int64(now-float64(int64(now)))
			c.Set("Retry-After", strconv.FormatInt(retry, 10))
			return c.Status(429).JSON(fiber.Map{"error": "rate limit exceeded"})
		}
		return c.Next()
	}
}

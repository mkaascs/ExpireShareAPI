package redis

import (
	"context"
	"expire-share/internal/config"
	"fmt"
	"github.com/redis/go-redis/v9"
)

const prefix = "rate"

type RateLimiter struct {
	client *redis.Client
	cfg    config.RateLimiter
}

func NewRateLimiter(client *redis.Client, cfg config.RateLimiter) *RateLimiter {
	return &RateLimiter{client: client, cfg: cfg}
}

func (rl *RateLimiter) Allow(ctx context.Context, field, value string) (bool, error) {
	const fn = "infrastructure.redis.RateLimiter.Allow"
	key := fmt.Sprintf("%s:%s:%s", prefix, field, value)

	count, err := rl.client.Incr(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("%s: failed to incr key: %w", fn, err)
	}

	if count == 1 {
		rl.client.Expire(ctx, key, rl.cfg.Window)
	}

	if count > int64(rl.cfg.MaxAttempts) {
		rl.client.Expire(ctx, key, rl.cfg.BlockDuration)
		return false, nil
	}

	return true, nil
}

func (rl *RateLimiter) Reset(ctx context.Context, field, value string) error {
	const fn = "infrastructure.redis.RateLimiter.Reset"
	key := fmt.Sprintf("%s:%s:%s", prefix, field, value)

	if _, err := rl.client.Del(ctx, key).Result(); err != nil {
		return fmt.Errorf("%s: failed to del key: %w", fn, err)
	}

	return nil
}

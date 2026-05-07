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
	params config.RateLimiterParams
}

func NewRateLimiter(client *redis.Client, params config.RateLimiterParams) *RateLimiter {
	return &RateLimiter{client: client, params: params}
}

var allowScript = redis.NewScript(`
local key = KEYS[1]
local max = tonumber(ARGV[1])
local window = tonumber(ARGV[2])
local block = tonumber(ARGV[3])

local count = redis.call("INCR", key)

if count == 1 then
    redis.call("EXPIRE", key, window)
end

if count > max then
    redis.call("EXPIRE", key, block)
    return 0
end

return 1
`)

func (rl *RateLimiter) Allow(ctx context.Context, value string) (bool, error) {
	const fn = "infrastructure.redis.RateLimiter.Allow"
	key := fmt.Sprintf("%s:%s:%s", prefix, rl.params.Field, value)

	result, err := allowScript.Run(ctx, rl.client,
		[]string{key},
		rl.params.MaxAttempts,
		int(rl.params.Window.Seconds()),
		int(rl.params.BlockDuration.Seconds()),
	).Int()

	if err != nil {
		return false, fmt.Errorf("%s: %w", fn, err)
	}

	return result == 1, nil
}

func (rl *RateLimiter) Reset(ctx context.Context, value string) error {
	const fn = "infrastructure.redis.RateLimiter.Reset"
	key := fmt.Sprintf("%s:%s:%s", prefix, rl.params.Field, value)

	if _, err := rl.client.Del(ctx, key).Result(); err != nil {
		return fmt.Errorf("%s: failed to del key: %w", fn, err)
	}

	return nil
}

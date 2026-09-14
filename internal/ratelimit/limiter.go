// Package ratelimit implements a per-user/per-server sliding-window rate limiter
// backed by Redis sorted sets. All operations are atomic via a Lua script.
package ratelimit

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/mcp-gate/mcp-gate/internal/cache"
)

// slidingWindowScript is an atomic Lua script that:
//  1. Removes entries older than the window.
//  2. Adds the current request (timestamp as score, unique nanosecond key as member).
//  3. Counts entries in the window.
//  4. Sets the key TTL so idle keys expire cleanly.
//  5. Returns the current count.
//
// KEYS[1] = rate limit key
// ARGV[1] = current time as Unix nanoseconds (string)
// ARGV[2] = window cutoff as Unix nanoseconds (string)  — (now - window)
// ARGV[3] = TTL in seconds for the key
var slidingWindowScript = redis.NewScript(`
local key    = KEYS[1]
local now    = ARGV[1]
local cutoff = ARGV[2]
local ttl    = tonumber(ARGV[3])

-- Remove requests outside the sliding window.
redis.call('ZREMRANGEBYSCORE', key, '-inf', cutoff)

-- Record the current request. Using nanoseconds makes member names unique.
redis.call('ZADD', key, now, now)

-- Count requests in the current window.
local count = redis.call('ZCARD', key)

-- Refresh TTL so idle keys don't live forever.
redis.call('EXPIRE', key, ttl)

return count
`)

// Limiter checks and records rate-limit state in Redis.
type Limiter struct {
	redis *redis.Client
}

// New creates a Limiter.
func New(rdb *redis.Client) *Limiter {
	return &Limiter{redis: rdb}
}

// Result is returned by Allow.
type Result struct {
	// Allowed is false when the user has exceeded their quota.
	Allowed bool
	// Count is the number of requests in the current window.
	Count int64
	// Limit is the configured cap.
	Limit int32
}

// Allow records the current request and returns whether it is within the limit.
// maxCallsPerMinute=0 is treated as "deny all" (used for explicitly blocked tools).
// If Redis is unavailable, it fails closed (returns not allowed) and logs the error.
func (l *Limiter) Allow(ctx context.Context, userID, serverID string, maxCallsPerMinute int32) (Result, error) {
	if maxCallsPerMinute <= 0 {
		return Result{Allowed: false, Limit: maxCallsPerMinute}, nil
	}

	key := cache.RateLimitKey(userID, serverID)
	now := time.Now().UnixNano()
	windowNano := int64(cache.RateLimitWindow) * int64(time.Second)
	cutoff := now - windowNano
	ttl := cache.RateLimitWindow + 5 // a few extra seconds so the key outlives the window

	count, err := slidingWindowScript.Run(ctx, l.redis, []string{key},
		fmt.Sprintf("%d", now),
		fmt.Sprintf("%d", cutoff),
		fmt.Sprintf("%d", ttl),
	).Int64()

	if err != nil {
		// Redis unavailable — fail closed.
		slog.ErrorContext(ctx, "rate limiter Redis error; failing closed",
			"user_id", userID, "server_id", serverID, "err", err)
		return Result{Allowed: false, Limit: maxCallsPerMinute}, err
	}

	allowed := count <= int64(maxCallsPerMinute)
	return Result{
		Allowed: allowed,
		Count:   count,
		Limit:   maxCallsPerMinute,
	}, nil
}

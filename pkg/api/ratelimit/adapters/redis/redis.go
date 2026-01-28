// Package redis provides a Redis-backed distributed rate limiter.
package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/algorithms/ratelimit"
	goredis "github.com/redis/go-redis/v9"
)

// Strategy defines the rate limiting algorithm.
type Strategy int

const (
	StrategyTokenBucket Strategy = iota
	StrategyLeakyBucket
	StrategyFixedWindow
	StrategySlidingWindow
)

// DistributedLimiter uses Redis Lua scripts for atomic rate limiting.
// This ensures correctness across multiple application instances.
type DistributedLimiter struct {
	client   goredis.Cmdable
	strategy Strategy
}

// New creates a new distributed rate limiter.
func New(client goredis.Cmdable, strategy Strategy) *DistributedLimiter {
	return &DistributedLimiter{
		client:   client,
		strategy: strategy,
	}
}

func (l *DistributedLimiter) Allow(ctx context.Context, key string, limit int64, period time.Duration) (*ratelimit.Result, error) {
	switch l.strategy {
	case StrategyTokenBucket:
		return l.tokenBucketAllow(ctx, key, limit, period)
	case StrategySlidingWindow:
		return l.slidingWindowAllow(ctx, key, limit, period)
	default:
		return l.fixedWindowAllow(ctx, key, limit, period)
	}
}

// =========================================================================
// Fixed Window - Redis Lua Script
// =========================================================================

var fixedWindowScript = goredis.NewScript(`
local key = KEYS[1]
local limit = tonumber(ARGV[1])
local period = tonumber(ARGV[2])

local current = redis.call('INCR', key)
if current == 1 then
    redis.call('EXPIRE', key, period)
end

local remaining = limit - current
if remaining < 0 then
    remaining = 0
end

local ttl = redis.call('TTL', key)
if ttl < 0 then
    ttl = period
end

if current <= limit then
    return {1, remaining, ttl}
else
    return {0, remaining, ttl}
end
`)

func (l *DistributedLimiter) fixedWindowAllow(ctx context.Context, key string, limit int64, period time.Duration) (*ratelimit.Result, error) {
	cacheKey := fmt.Sprintf("rl:dist:fixed:%s", key)

	result, err := fixedWindowScript.Run(ctx, l.client, []string{cacheKey}, limit, int64(period.Seconds())).Int64Slice()
	if err != nil {
		return nil, err
	}

	return &ratelimit.Result{
		Allowed:   result[0] == 1,
		Remaining: result[1],
		Reset:     time.Duration(result[2]) * time.Second,
	}, nil
}

// =========================================================================
// Token Bucket - Redis Lua Script
// =========================================================================

var tokenBucketScript = goredis.NewScript(`
local key = KEYS[1]
local capacity = tonumber(ARGV[1])
local refill_rate = tonumber(ARGV[2])
local now = tonumber(ARGV[3])
local ttl = tonumber(ARGV[4])

local data = redis.call('HMGET', key, 'tokens', 'last_refill')
local tokens = tonumber(data[1]) or capacity
local last_refill = tonumber(data[2]) or now

local elapsed = (now - last_refill) / 1000.0
local tokens_to_add = elapsed * refill_rate
tokens = math.min(capacity, tokens + tokens_to_add)

local allowed = 0
local remaining = tokens
if tokens >= 1 then
    tokens = tokens - 1
    remaining = tokens
    allowed = 1
end

redis.call('HMSET', key, 'tokens', tokens, 'last_refill', now)
redis.call('EXPIRE', key, ttl)

local reset_ms = 0
if allowed == 0 then
    reset_ms = math.ceil((1 - tokens) / refill_rate * 1000)
end

return {allowed, math.floor(remaining), reset_ms}
`)

func (l *DistributedLimiter) tokenBucketAllow(ctx context.Context, key string, limit int64, period time.Duration) (*ratelimit.Result, error) {
	cacheKey := fmt.Sprintf("rl:dist:tb:%s", key)
	refillRate := float64(limit) / period.Seconds()
	now := time.Now().UnixMilli()
	ttl := int64(period.Seconds() * 2)

	result, err := tokenBucketScript.Run(ctx, l.client, []string{cacheKey}, limit, refillRate, now, ttl).Int64Slice()
	if err != nil {
		return nil, err
	}

	return &ratelimit.Result{
		Allowed:   result[0] == 1,
		Remaining: result[1],
		Reset:     time.Duration(result[2]) * time.Millisecond,
	}, nil
}

// =========================================================================
// Sliding Window Log - Redis Lua Script
// =========================================================================

var slidingWindowScript = goredis.NewScript(`
local key = KEYS[1]
local limit = tonumber(ARGV[1])
local window_ms = tonumber(ARGV[2])
local now = tonumber(ARGV[3])
local request_id = ARGV[4]

local window_start = now - window_ms
redis.call('ZREMRANGEBYSCORE', key, '-inf', window_start)

local count = redis.call('ZCARD', key)

if count < limit then
    redis.call('ZADD', key, now, request_id)
    redis.call('PEXPIRE', key, window_ms)
    
    local remaining = limit - count - 1
    return {1, remaining, 0}
else
    local oldest = redis.call('ZRANGE', key, 0, 0, 'WITHSCORES')
    local reset_ms = 0
    if oldest and #oldest >= 2 then
        local oldest_time = tonumber(oldest[2])
        reset_ms = (oldest_time + window_ms) - now
        if reset_ms < 0 then reset_ms = 0 end
    end
    
    return {0, 0, reset_ms}
end
`)

func (l *DistributedLimiter) slidingWindowAllow(ctx context.Context, key string, limit int64, period time.Duration) (*ratelimit.Result, error) {
	cacheKey := fmt.Sprintf("rl:dist:slide:%s", key)
	now := time.Now().UnixMilli()
	requestID := fmt.Sprintf("%d:%d", now, time.Now().UnixNano()%1000000)

	result, err := slidingWindowScript.Run(ctx, l.client, []string{cacheKey}, limit, period.Milliseconds(), now, requestID).Int64Slice()
	if err != nil {
		return nil, err
	}

	return &ratelimit.Result{
		Allowed:   result[0] == 1,
		Remaining: result[1],
		Reset:     time.Duration(result[2]) * time.Millisecond,
	}, nil
}

var _ ratelimit.Limiter = (*DistributedLimiter)(nil)

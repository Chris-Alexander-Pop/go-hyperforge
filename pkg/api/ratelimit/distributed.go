package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// DistributedLimiter uses Redis Lua scripts for atomic rate limiting.
// This ensures correctness across multiple application instances.
type DistributedLimiter struct {
	client   redis.Cmdable
	strategy Strategy
}

// NewDistributedLimiter creates a new distributed rate limiter.
func NewDistributedLimiter(client redis.Cmdable, strategy Strategy) *DistributedLimiter {
	return &DistributedLimiter{
		client:   client,
		strategy: strategy,
	}
}

func (l *DistributedLimiter) Allow(ctx context.Context, key string, limit int64, period time.Duration) (*Result, error) {
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
// Atomic increment with expiration

var fixedWindowScript = redis.NewScript(`
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

func (l *DistributedLimiter) fixedWindowAllow(ctx context.Context, key string, limit int64, period time.Duration) (*Result, error) {
	cacheKey := fmt.Sprintf("rl:dist:fixed:%s", key)

	result, err := fixedWindowScript.Run(ctx, l.client, []string{cacheKey}, limit, int64(period.Seconds())).Int64Slice()
	if err != nil {
		return nil, err
	}

	return &Result{
		Allowed:   result[0] == 1,
		Remaining: result[1],
		Reset:     time.Duration(result[2]) * time.Second,
	}, nil
}

// =========================================================================
// Token Bucket - Redis Lua Script
// =========================================================================
// Atomic token bucket with refill calculation

var tokenBucketScript = redis.NewScript(`
local key = KEYS[1]
local capacity = tonumber(ARGV[1])     -- max tokens (limit)
local refill_rate = tonumber(ARGV[2])  -- tokens per second
local now = tonumber(ARGV[3])          -- current timestamp in milliseconds
local ttl = tonumber(ARGV[4])          -- key expiration in seconds

-- Get current state
local data = redis.call('HMGET', key, 'tokens', 'last_refill')
local tokens = tonumber(data[1]) or capacity
local last_refill = tonumber(data[2]) or now

-- Calculate tokens to add
local elapsed = (now - last_refill) / 1000.0  -- convert ms to seconds
local tokens_to_add = elapsed * refill_rate
tokens = math.min(capacity, tokens + tokens_to_add)

-- Try to consume a token
local allowed = 0
local remaining = tokens
if tokens >= 1 then
    tokens = tokens - 1
    remaining = tokens
    allowed = 1
end

-- Save state
redis.call('HMSET', key, 'tokens', tokens, 'last_refill', now)
redis.call('EXPIRE', key, ttl)

-- Calculate time until next token (if denied)
local reset_ms = 0
if allowed == 0 then
    reset_ms = math.ceil((1 - tokens) / refill_rate * 1000)
end

return {allowed, math.floor(remaining), reset_ms}
`)

func (l *DistributedLimiter) tokenBucketAllow(ctx context.Context, key string, limit int64, period time.Duration) (*Result, error) {
	cacheKey := fmt.Sprintf("rl:dist:tb:%s", key)
	refillRate := float64(limit) / period.Seconds()
	now := time.Now().UnixMilli()
	ttl := int64(period.Seconds() * 2)

	result, err := tokenBucketScript.Run(ctx, l.client, []string{cacheKey}, limit, refillRate, now, ttl).Int64Slice()
	if err != nil {
		return nil, err
	}

	return &Result{
		Allowed:   result[0] == 1,
		Remaining: result[1],
		Reset:     time.Duration(result[2]) * time.Millisecond,
	}, nil
}

// =========================================================================
// Sliding Window Log - Redis Lua Script
// =========================================================================
// Uses sorted set to track individual requests with timestamps

var slidingWindowScript = redis.NewScript(`
local key = KEYS[1]
local limit = tonumber(ARGV[1])
local window_ms = tonumber(ARGV[2])  -- window size in milliseconds
local now = tonumber(ARGV[3])        -- current timestamp in milliseconds
local request_id = ARGV[4]           -- unique request identifier

-- Remove old entries outside the window
local window_start = now - window_ms
redis.call('ZREMRANGEBYSCORE', key, '-inf', window_start)

-- Count current requests in window
local count = redis.call('ZCARD', key)

if count < limit then
    -- Add new request
    redis.call('ZADD', key, now, request_id)
    redis.call('PEXPIRE', key, window_ms)
    
    local remaining = limit - count - 1
    return {1, remaining, 0}
else
    -- Get oldest entry to calculate when it expires
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

func (l *DistributedLimiter) slidingWindowAllow(ctx context.Context, key string, limit int64, period time.Duration) (*Result, error) {
	cacheKey := fmt.Sprintf("rl:dist:slide:%s", key)
	now := time.Now().UnixMilli()
	requestID := fmt.Sprintf("%d:%d", now, time.Now().UnixNano()%1000000)

	result, err := slidingWindowScript.Run(ctx, l.client, []string{cacheKey}, limit, period.Milliseconds(), now, requestID).Int64Slice()
	if err != nil {
		return nil, err
	}

	return &Result{
		Allowed:   result[0] == 1,
		Remaining: result[1],
		Reset:     time.Duration(result[2]) * time.Millisecond,
	}, nil
}

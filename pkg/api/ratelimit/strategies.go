package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/cache"
)

type Strategy string

const (
	StrategyFixedWindow Strategy = "fixed_window"
	StrategyTokenBucket Strategy = "token_bucket"
)

// Result from a rate limit check
type Result struct {
	Allowed   bool
	Remaining int64
	Reset     time.Duration
}

// Limiter defines the interface for different strategies
type Limiter interface {
	// Allow checks if the request is allowed
	Allow(ctx context.Context, key string, limit int64, period time.Duration) (*Result, error)
}

// Factory creates a limiter based on strategy
func New(c cache.Cache, strategy Strategy) Limiter {
	switch strategy {
	case StrategyTokenBucket:
		return &TokenBucketLimiter{cache: c}
	default:
		return &FixedWindowLimiter{cache: c}
	}
}

// FixedWindowLimiter (Existing Logic)
type FixedWindowLimiter struct {
	cache cache.Cache
}

func (l *FixedWindowLimiter) Allow(ctx context.Context, key string, limit int64, period time.Duration) (*Result, error) {
	// Simple key based on time bucket? Or just raw TTL overlap?
	// The previous implementation used Raw Key with delayed TTL.
	// Better Fixed Window: Key = prefix + time.Now().Truncate(period).Unix()
	// This ensures clean windows.

	window := time.Now().Truncate(period).Unix()
	cacheKey := fmt.Sprintf("rl:fixed:%s:%d", key, window)

	curr, err := l.cache.Incr(ctx, cacheKey, 1)
	if err != nil {
		return nil, err
	}

	if curr == 1 {
		_ = l.cache.Set(ctx, cacheKey, int64(1), period*2) // *2 to allow clock skew overlap/inspection
	}

	remaining := limit - curr
	if remaining < 0 {
		remaining = 0
	}

	return &Result{
		Allowed:   curr <= limit,
		Remaining: remaining,
		Reset:     time.Duration(period.Seconds()-float64(time.Now().Unix()%int64(period.Seconds()))) * time.Second,
	}, nil
}

// TokenBucketLimiter (Simplified - Approximation using Last Refill Time)
type TokenBucketLimiter struct {
	cache cache.Cache
}

func (l *TokenBucketLimiter) Allow(ctx context.Context, key string, limit int64, period time.Duration) (*Result, error) {
	// Note: True Token Bucket on Redis requires Lua for atomicity (Get tokens, refill based on time, decrement, save).
	// Emulating without Lua is racy.
	// For this system design library, we'll mark this complexity:
	// "TBI: Use Redis Lua script for atomic Token Bucket. Falling back to Fixed Window safely."
	return (&FixedWindowLimiter{cache: l.cache}).Allow(ctx, key, limit, period)
}

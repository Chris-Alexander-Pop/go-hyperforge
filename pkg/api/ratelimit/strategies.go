package ratelimit

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/cache"
)

type Strategy string

const (
	StrategyFixedWindow   Strategy = "fixed_window"
	StrategyTokenBucket   Strategy = "token_bucket"
	StrategyLeakyBucket   Strategy = "leaky_bucket"
	StrategySlidingWindow Strategy = "sliding_window"
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
		return NewTokenBucketLimiter(c)
	case StrategyLeakyBucket:
		return NewLeakyBucketLimiter(c)
	case StrategySlidingWindow:
		return NewSlidingWindowLimiter(c)
	default:
		return &FixedWindowLimiter{cache: c}
	}
}

// =========================================================================
// Fixed Window Rate Limiter
// =========================================================================
// Simple time-bucketed counter. Resets at the start of each window.
// Pros: Simple, low memory
// Cons: Burst at window boundaries (2x traffic possible)

type FixedWindowLimiter struct {
	cache cache.Cache
}

func (l *FixedWindowLimiter) Allow(ctx context.Context, key string, limit int64, period time.Duration) (*Result, error) {
	window := time.Now().Truncate(period).Unix()
	cacheKey := fmt.Sprintf("rl:fixed:%s:%d", key, window)

	curr, err := l.cache.Incr(ctx, cacheKey, 1)
	if err != nil {
		return nil, err
	}

	if curr == 1 {
		_ = l.cache.Set(ctx, cacheKey, int64(1), period*2)
	}

	remaining := limit - curr
	if remaining < 0 {
		remaining = 0
	}

	resetSeconds := period.Seconds() - float64(time.Now().Unix()%int64(period.Seconds()))

	return &Result{
		Allowed:   curr <= limit,
		Remaining: remaining,
		Reset:     time.Duration(resetSeconds) * time.Second,
	}, nil
}

// =========================================================================
// Token Bucket Rate Limiter
// =========================================================================
// Bucket fills with tokens at a steady rate. Each request consumes a token.
// Pros: Allows bursts up to bucket capacity, smooth average rate
// Cons: More complex state management

type TokenBucketLimiter struct {
	cache  cache.Cache
	states sync.Map // In-memory state for non-distributed use
}

type tokenBucketState struct {
	tokens     float64
	lastRefill time.Time
	mu         sync.Mutex
}

func NewTokenBucketLimiter(c cache.Cache) *TokenBucketLimiter {
	return &TokenBucketLimiter{cache: c}
}

func (l *TokenBucketLimiter) Allow(ctx context.Context, key string, limit int64, period time.Duration) (*Result, error) {
	// For in-memory token bucket, we track state locally
	// For distributed, see distributed.go with Redis Lua scripts

	stateKey := fmt.Sprintf("tb:%s", key)

	// Get or create state
	val, _ := l.states.LoadOrStore(stateKey, &tokenBucketState{
		tokens:     float64(limit),
		lastRefill: time.Now(),
	})
	state := val.(*tokenBucketState)

	state.mu.Lock()
	defer state.mu.Unlock()

	// Calculate tokens to add based on elapsed time
	now := time.Now()
	elapsed := now.Sub(state.lastRefill)
	refillRate := float64(limit) / period.Seconds() // tokens per second
	tokensToAdd := elapsed.Seconds() * refillRate

	// Refill bucket (capped at limit)
	state.tokens += tokensToAdd
	if state.tokens > float64(limit) {
		state.tokens = float64(limit)
	}
	state.lastRefill = now

	// Try to consume a token
	if state.tokens >= 1 {
		state.tokens--
		return &Result{
			Allowed:   true,
			Remaining: int64(state.tokens),
			Reset:     time.Duration(1/refillRate) * time.Second,
		}, nil
	}

	// Calculate time until next token
	timeUntilToken := time.Duration((1 - state.tokens) / refillRate * float64(time.Second))

	return &Result{
		Allowed:   false,
		Remaining: 0,
		Reset:     timeUntilToken,
	}, nil
}

// =========================================================================
// Leaky Bucket Rate Limiter
// =========================================================================
// Requests "leak" out at a constant rate. Smoothest traffic pattern.
// Pros: Perfect smooth rate, no bursts
// Cons: Bursts are queued or rejected, more latency

type LeakyBucketLimiter struct {
	cache   cache.Cache
	buckets sync.Map
}

type leakyBucketState struct {
	queue    int64     // Current queue size
	lastLeak time.Time // Last time we processed a request
	mu       sync.Mutex
}

func NewLeakyBucketLimiter(c cache.Cache) *LeakyBucketLimiter {
	return &LeakyBucketLimiter{cache: c}
}

func (l *LeakyBucketLimiter) Allow(ctx context.Context, key string, limit int64, period time.Duration) (*Result, error) {
	stateKey := fmt.Sprintf("lb:%s", key)

	val, _ := l.buckets.LoadOrStore(stateKey, &leakyBucketState{
		queue:    0,
		lastLeak: time.Now(),
	})
	state := val.(*leakyBucketState)

	state.mu.Lock()
	defer state.mu.Unlock()

	now := time.Now()
	leakRate := float64(limit) / period.Seconds() // requests per second

	// Calculate how many requests have "leaked" since last check
	elapsed := now.Sub(state.lastLeak)
	leaked := int64(elapsed.Seconds() * leakRate)

	// Drain the queue
	state.queue -= leaked
	if state.queue < 0 {
		state.queue = 0
	}
	state.lastLeak = now

	// Check if we can add to queue (queue size = burst capacity = limit)
	if state.queue < limit {
		state.queue++
		return &Result{
			Allowed:   true,
			Remaining: limit - state.queue,
			Reset:     time.Duration(1/leakRate) * time.Second,
		}, nil
	}

	// Queue is full
	return &Result{
		Allowed:   false,
		Remaining: 0,
		Reset:     time.Duration(1/leakRate) * time.Second,
	}, nil
}

// =========================================================================
// Sliding Window Rate Limiter
// =========================================================================
// Weighted combination of current and previous window.
// Pros: More accurate than fixed window, prevents boundary bursts
// Cons: Slightly more computation

type SlidingWindowLimiter struct {
	cache cache.Cache
}

func NewSlidingWindowLimiter(c cache.Cache) *SlidingWindowLimiter {
	return &SlidingWindowLimiter{cache: c}
}

func (l *SlidingWindowLimiter) Allow(ctx context.Context, key string, limit int64, period time.Duration) (*Result, error) {
	now := time.Now()

	// Current window
	currentWindow := now.Truncate(period).Unix()
	previousWindow := currentWindow - int64(period.Seconds())

	currentKey := fmt.Sprintf("rl:slide:%s:%d", key, currentWindow)
	previousKey := fmt.Sprintf("rl:slide:%s:%d", key, previousWindow)

	// Get counts from both windows
	var currentCount, previousCount int64

	if err := l.cache.Get(ctx, currentKey, &currentCount); err != nil {
		currentCount = 0
	}
	if err := l.cache.Get(ctx, previousKey, &previousCount); err != nil {
		previousCount = 0
	}

	// Calculate weighted count
	// Weight = how far into the current window we are
	windowProgress := float64(now.UnixMilli()%int64(period.Milliseconds())) / float64(period.Milliseconds())
	previousWeight := 1.0 - windowProgress

	weightedCount := float64(currentCount) + (float64(previousCount) * previousWeight)

	if weightedCount >= float64(limit) {
		remaining := limit - int64(weightedCount)
		if remaining < 0 {
			remaining = 0
		}
		return &Result{
			Allowed:   false,
			Remaining: remaining,
			Reset:     period - time.Duration(float64(period)*windowProgress),
		}, nil
	}

	// Increment current window
	newCount, err := l.cache.Incr(ctx, currentKey, 1)
	if err != nil {
		return nil, err
	}

	if newCount == 1 {
		_ = l.cache.Set(ctx, currentKey, int64(1), period*2)
	}

	remaining := limit - int64(weightedCount) - 1
	if remaining < 0 {
		remaining = 0
	}

	return &Result{
		Allowed:   true,
		Remaining: remaining,
		Reset:     period - time.Duration(float64(period)*windowProgress),
	}, nil
}

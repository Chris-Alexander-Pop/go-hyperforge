package ratelimit

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/cache"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
)

// Strategy defines the rate limiting algorithm.
type Strategy int

const (
	StrategyTokenBucket Strategy = iota
	StrategyLeakyBucket
	StrategyFixedWindow
	StrategySlidingWindow
)

// Result is the result of a limit check.
type Result struct {
	Allowed   bool
	Remaining int64
	Reset     time.Duration
}

// Limiter determines if an action is allowed.
type Limiter interface {
	// Allow checks if the key is allowed to perform 'cost' operations.
	// period is only relevant for window-based strategies.
	Allow(ctx context.Context, key string, limit int64, period time.Duration) (*Result, error)
}

// SlidingWindowLimiter implements a sliding window counter using Redis/Cache.
type SlidingWindowLimiter struct {
	store cache.Cache
}

func (l *SlidingWindowLimiter) Allow(ctx context.Context, key string, limit int64, period time.Duration) (*Result, error) {
	// We use a simplified specific implementation here, leveraging the cache interface.
	// Ideally, this should use atomic operations (Lua script in Redis).
	// Since Cache interface is generic, we'll implement a rough approximation or assume the cache supports it.

	// For production, this needs true sliding window logic (ZSET in Redis).
	// Here we will implement a Fixed Window approximation if the store is simple,
	// or try to do better if we can.

	// Let's implement generic Fixed Window for v1 compatibility with generic cache
	now := time.Now()
	windowKey := key + ":" + now.Truncate(period).Format(time.RFC3339)

	count, err := l.store.Incr(ctx, windowKey, 1)
	if err != nil {
		return nil, errors.Wrap(err, "ratelimit error")
	}

	// Set TTL if new
	if count == 1 {
		_ = l.store.Set(ctx, windowKey, 1, period)
	}

	remaining := limit - count
	if remaining < 0 {
		remaining = 0
	}

	reset := time.Until(now.Truncate(period).Add(period))

	return &Result{
		Allowed:   count <= limit,
		Remaining: remaining,
		Reset:     reset,
	}, nil
}

// InMemLimiter is a simple thread-safe in-memory limiter (Token Bucket).
// Useful for single-instance applications or sidecars.
type InMemLimiter struct {
	rate       float64 // tokens per second
	burst      int64
	tokens     map[string]float64
	lastUpdate map[string]time.Time
	mu         sync.Mutex
}

func NewInMemLimiter(rate float64, burst int64) *InMemLimiter {
	return &InMemLimiter{
		rate:       rate,
		burst:      burst,
		tokens:     make(map[string]float64),
		lastUpdate: make(map[string]time.Time),
	}
}

func (l *InMemLimiter) Allow(ctx context.Context, key string, limit int64, period time.Duration) (*Result, error) {
	// Adapt generic Allow signature to InMemLimiter logic
	// InMemLimiter struct above has fixed rate/burst, but Allow interface passes them in.
	// For API compatibility, we might need to adjust or ignore the struct fields if we use the generic interface.
	// However, InMemLimiter usually has fixed capacity.
	// Let's implement the simpler Allow(key) for internal use, and interface compliance.

	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()

	tokens, exists := l.tokens[key]
	if !exists {
		tokens = float64(l.burst)
		l.lastUpdate[key] = now
	} else {
		elapsed := now.Sub(l.lastUpdate[key]).Seconds()
		tokens += elapsed * l.rate
		if tokens > float64(l.burst) {
			tokens = float64(l.burst)
		}
		l.lastUpdate[key] = now
	}

	if tokens >= 1 {
		l.tokens[key] = tokens - 1
		return &Result{Allowed: true, Remaining: int64(tokens - 1)}, nil
	}

	return &Result{Allowed: false, Remaining: 0}, nil
}

// FixedWindowLimiter implements a simple time-bucketed counter.
type FixedWindowLimiter struct {
	store cache.Cache
}

func (l *FixedWindowLimiter) Allow(ctx context.Context, key string, limit int64, period time.Duration) (*Result, error) {
	window := time.Now().Truncate(period).Unix()
	cacheKey := fmt.Sprintf("rl:fixed:%s:%d", key, window)

	curr, err := l.store.Incr(ctx, cacheKey, 1)
	if err != nil {
		return nil, errors.Wrap(err, "ratelimit error")
	}

	if curr == 1 {
		_ = l.store.Set(ctx, cacheKey, int64(1), period*2)
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

// TokenBucketLimiter implements a distributed token bucket.
// Note: This naive implementation uses a simplified locking strategy via cache/store interactions
// which may face race conditions in high concurrency without Lua scripts.
type TokenBucketLimiter struct {
	store  cache.Cache
	states sync.Map // Local cache of states to reduce read load, assuming stickiness or acceptable imprecision
}

// tokenBucketState for local tracking to reduce cache blasts
type tokenBucketState struct {
	tokens     float64
	lastRefill time.Time
	mu         sync.Mutex
}

func (l *TokenBucketLimiter) Allow(ctx context.Context, key string, limit int64, period time.Duration) (*Result, error) {
	// For distributed token bucket without Lua, we often need to rely on local state
	// or perform multiple round trips (CAS).
	// Here we use the local state approach similar to api/ratelimit for performance,
	// acknowledging this is per-instance limiting if not utilizing a shared store correctly.
	// If 'store' is Redis, true distributed TB needs Lua.

	stateKey := fmt.Sprintf("tb:%s", key)
	val, _ := l.states.LoadOrStore(stateKey, &tokenBucketState{
		tokens:     float64(limit),
		lastRefill: time.Now(),
	})
	state := val.(*tokenBucketState)

	state.mu.Lock()
	defer state.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(state.lastRefill)
	refillRate := float64(limit) / period.Seconds()
	tokensToAdd := elapsed.Seconds() * refillRate

	state.tokens += tokensToAdd
	if state.tokens > float64(limit) {
		state.tokens = float64(limit)
	}
	state.lastRefill = now

	if state.tokens >= 1 {
		state.tokens--
		return &Result{
			Allowed:   true,
			Remaining: int64(state.tokens),
			Reset:     time.Duration(1/refillRate) * time.Second,
		}, nil
	}

	timeUntilToken := time.Duration((1 - state.tokens) / refillRate * float64(time.Second))
	return &Result{
		Allowed:   false,
		Remaining: 0,
		Reset:     timeUntilToken,
	}, nil
}

// LeakyBucketLimiter implements a generic leaky bucket.
type LeakyBucketLimiter struct {
	store   cache.Cache
	buckets sync.Map
}

type leakyBucketState struct {
	queue    int64
	lastLeak time.Time
	mu       sync.Mutex
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
	leakRate := float64(limit) / period.Seconds()

	elapsed := now.Sub(state.lastLeak)
	leaked := int64(elapsed.Seconds() * leakRate)

	state.queue -= leaked
	if state.queue < 0 {
		state.queue = 0
	}
	state.lastLeak = now

	if state.queue < limit {
		state.queue++
		return &Result{
			Allowed:   true,
			Remaining: limit - state.queue,
			Reset:     time.Duration(1/leakRate) * time.Second,
		}, nil
	}

	return &Result{
		Allowed:   false,
		Remaining: 0,
		Reset:     time.Duration(1/leakRate) * time.Second,
	}, nil
}

// Factory
func New(store cache.Cache, strategy Strategy) Limiter {
	switch strategy {
	case StrategyTokenBucket:
		return &TokenBucketLimiter{store: store}
	case StrategyLeakyBucket:
		return &LeakyBucketLimiter{store: store}
	case StrategyFixedWindow:
		return &FixedWindowLimiter{store: store}
	case StrategySlidingWindow:
		fallthrough
	default:
		return &SlidingWindowLimiter{store: store}
	}
}

package tokenbucket

import (
	"context"
	"sync"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/algorithms/ratelimit"
	"github.com/chris-alexander-pop/system-design-library/pkg/cache"
	"github.com/chris-alexander-pop/system-design-library/pkg/concurrency"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
)

// DistLimiter implements a distributed token bucket backed by cache.Cache.
// Token state (remaining tokens + last refill time) is stored under "tb:"+key
// so multiple processes sharing the same store observe the same budget.
//
// Concurrent writers still need a coherent store (single-key CAS / Lua is
// ideal); this implementation serializes per-key within a process and uses
// Get/Set on the shared store for cross-process visibility.
type DistLimiter struct {
	store  cache.Cache
	states sync.Map // key -> *concurrency.SmartMutex (process-local lock)
}

type distBucketState struct {
	Tokens     float64 `json:"tokens"`
	LastRefill int64   `json:"last_refill"` // unix nanoseconds
}

// NewDist creates a new distributed TokenBucket limiter.
func NewDist(store cache.Cache) *DistLimiter {
	return &DistLimiter{store: store}
}

func (l *DistLimiter) keyMutex(key string) *concurrency.SmartMutex {
	if v, ok := l.states.Load(key); ok {
		return v.(*concurrency.SmartMutex)
	}
	mu := concurrency.NewSmartMutex(concurrency.MutexConfig{Name: "tokenbucket-dist:" + key})
	actual, _ := l.states.LoadOrStore(key, mu)
	return actual.(*concurrency.SmartMutex)
}

func (l *DistLimiter) Allow(ctx context.Context, key string, limit int64, period time.Duration) (*ratelimit.Result, error) {
	if l.store == nil {
		return nil, errors.InvalidArgument("DistLimiter requires a non-nil cache store", nil)
	}
	if limit <= 0 || period <= 0 {
		return nil, errors.InvalidArgument("limit and period must be positive", nil)
	}

	mu := l.keyMutex(key)
	mu.Lock()
	defer mu.Unlock()

	storeKey := "tb:" + key
	now := time.Now()
	refillRate := float64(limit) / period.Seconds()

	var state distBucketState
	err := l.store.Get(ctx, storeKey, &state)
	if err != nil {
		if !cache.IsNotFound(err) {
			return nil, errors.Wrap(err, "tokenbucket dist get")
		}
		state = distBucketState{
			Tokens:     float64(limit),
			LastRefill: now.UnixNano(),
		}
	}

	last := time.Unix(0, state.LastRefill)
	elapsed := now.Sub(last).Seconds()
	if elapsed < 0 {
		elapsed = 0
	}
	state.Tokens += elapsed * refillRate
	if state.Tokens > float64(limit) {
		state.Tokens = float64(limit)
	}
	state.LastRefill = now.UnixNano()

	ttl := period * 2
	if state.Tokens >= 1 {
		state.Tokens--
		if err := l.store.Set(ctx, storeKey, state, ttl); err != nil {
			return nil, errors.Wrap(err, "tokenbucket dist set")
		}
		return &ratelimit.Result{
			Allowed:   true,
			Remaining: int64(state.Tokens),
			Reset:     time.Duration((1 / refillRate) * float64(time.Second)),
		}, nil
	}

	if err := l.store.Set(ctx, storeKey, state, ttl); err != nil {
		return nil, errors.Wrap(err, "tokenbucket dist set")
	}
	timeUntilToken := time.Duration((1 - state.Tokens) / refillRate * float64(time.Second))
	return &ratelimit.Result{
		Allowed:   false,
		Remaining: 0,
		Reset:     timeUntilToken,
	}, nil
}

// InMemoryLimiter is a simple thread-safe in-memory limiter.
type InMemoryLimiter struct {
	rate       float64
	burst      int64
	tokens     map[string]float64
	lastUpdate map[string]time.Time
	mu         *concurrency.SmartMutex
}

// NewInMemory creates a new in-memory TokenBucket limiter.
func NewInMemory(rate float64, burst int64) *InMemoryLimiter {
	return &InMemoryLimiter{
		rate:       rate,
		burst:      burst,
		tokens:     make(map[string]float64),
		lastUpdate: make(map[string]time.Time),
		mu:         concurrency.NewSmartMutex(concurrency.MutexConfig{Name: "inmemory-tokenbucket"}),
	}
}

func (l *InMemoryLimiter) Allow(ctx context.Context, key string, limit int64, period time.Duration) (*ratelimit.Result, error) {
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
		return &ratelimit.Result{Allowed: true, Remaining: int64(tokens - 1)}, nil
	}

	return &ratelimit.Result{Allowed: false, Remaining: 0}, nil
}

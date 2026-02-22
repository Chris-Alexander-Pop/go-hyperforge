package tokenbucket

import (
	"context"
	"sync"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/algorithms/ratelimit"
	"github.com/chris-alexander-pop/system-design-library/pkg/cache"
	"github.com/chris-alexander-pop/system-design-library/pkg/concurrency"
)

// DistLimiter implements a distributed token bucket.
type DistLimiter struct {
	store  cache.Cache
	states sync.Map
}

// tokenBucketState for local tracking
type tokenBucketState struct {
	tokens     float64
	lastRefill time.Time
	mu         *concurrency.SmartMutex
}

// NewDist creates a new distributed TokenBucket limiter.
func NewDist(store cache.Cache) *DistLimiter {
	return &DistLimiter{store: store}
}

func (l *DistLimiter) Allow(ctx context.Context, key string, limit int64, period time.Duration) (*ratelimit.Result, error) {
	stateKey := "tb:" + key
	val, ok := l.states.Load(stateKey)
	if !ok {
		val, _ = l.states.LoadOrStore(stateKey, &tokenBucketState{
			tokens:     float64(limit),
			lastRefill: time.Now(),
			mu:         concurrency.NewSmartMutex(concurrency.MutexConfig{Name: "tokenbucket-state"}),
		})
	}
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
		return &ratelimit.Result{
			Allowed:   true,
			Remaining: int64(state.tokens),
			Reset:     time.Duration(1/refillRate) * time.Second,
		}, nil
	}

	timeUntilToken := time.Duration((1 - state.tokens) / refillRate * float64(time.Second))
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

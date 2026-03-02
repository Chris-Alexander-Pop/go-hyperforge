package leakybucket

import (
	"context"
	"sync"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/algorithms/ratelimit"
	"github.com/chris-alexander-pop/system-design-library/pkg/cache"
)

// Limiter implements a generic leaky bucket.
type Limiter struct {
	store   cache.Cache
	buckets sync.Map
}

type state struct {
	queue    int64
	lastLeak time.Time
	mu       sync.Mutex
}

// New creates a new LeakyBucket limiter.
func New(store cache.Cache) *Limiter {
	return &Limiter{store: store}
}

func (l *Limiter) Allow(ctx context.Context, key string, limit int64, period time.Duration) (*ratelimit.Result, error) {
	stateKey := "lb:" + key
	val, ok := l.buckets.Load(stateKey)
	if !ok {
		val, _ = l.buckets.LoadOrStore(stateKey, &state{
			queue:    0,
			lastLeak: time.Now(),
		})
	}
	s := val.(*state)

	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	leakRate := float64(limit) / period.Seconds()

	elapsed := now.Sub(s.lastLeak)
	leaked := int64(elapsed.Seconds() * leakRate)

	s.queue -= leaked
	if s.queue < 0 {
		s.queue = 0
	}
	s.lastLeak = now

	if s.queue < limit {
		s.queue++
		return &ratelimit.Result{
			Allowed:   true,
			Remaining: limit - s.queue,
			Reset:     time.Duration(1/leakRate) * time.Second,
		}, nil
	}

	return &ratelimit.Result{
		Allowed:   false,
		Remaining: 0,
		Reset:     time.Duration(1/leakRate) * time.Second,
	}, nil
}

package slidingwindow

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/concurrency"
)

// Local is a process-local sliding-window limiter used by mesh/API facades.
type Local struct {
	mu       *concurrency.SmartMutex
	requests []time.Time
	limit    int
	window   time.Duration
}

// NewLocal creates a local sliding-window limiter.
func NewLocal(limit int, window time.Duration) *Local {
	return &Local{
		requests: make([]time.Time, 0, limit),
		limit:    limit,
		window:   window,
		mu:       concurrency.NewSmartMutex(concurrency.MutexConfig{Name: "slidingwindow-local"}),
	}
}

func (sw *Local) cleanup() {
	threshold := time.Now().Add(-sw.window)
	valid := sw.requests[:0]
	for _, t := range sw.requests {
		if t.After(threshold) {
			valid = append(valid, t)
		}
	}
	sw.requests = valid
}

// Allow reports whether one request is allowed.
func (sw *Local) Allow() bool {
	return sw.AllowN(1)
}

// AllowN reports whether n requests are allowed.
func (sw *Local) AllowN(n int) bool {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	sw.cleanup()
	if len(sw.requests)+n <= sw.limit {
		now := time.Now()
		for i := 0; i < n; i++ {
			sw.requests = append(sw.requests, now)
		}
		return true
	}
	return false
}

// Wait blocks until a request is allowed or ctx is cancelled.
func (sw *Local) Wait(ctx context.Context) error {
	for {
		if sw.Allow() {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}
}

// Reserve returns how long until the oldest request exits the window.
func (sw *Local) Reserve() time.Duration {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	sw.cleanup()
	if len(sw.requests) < sw.limit {
		return 0
	}
	return time.Until(sw.requests[0].Add(sw.window))
}

// Tokens returns remaining capacity in the current window.
func (sw *Local) Tokens() float64 {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	sw.cleanup()
	return float64(sw.limit - len(sw.requests))
}

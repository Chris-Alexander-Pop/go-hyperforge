package tokenbucket

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/concurrency"
)

// Local is a process-local token bucket used by mesh/API facades.
// It is the single implementation for non-keyed in-memory token buckets.
type Local struct {
	mu         *concurrency.SmartMutex
	tokens     float64
	capacity   float64
	rate       float64
	lastUpdate time.Time
}

// NewLocal creates a local token bucket with the given capacity and refill rate.
func NewLocal(capacity int, rate float64) *Local {
	return &Local{
		tokens:     float64(capacity),
		capacity:   float64(capacity),
		rate:       rate,
		lastUpdate: time.Now(),
		mu:         concurrency.NewSmartMutex(concurrency.MutexConfig{Name: "tokenbucket-local"}),
	}
}

func (tb *Local) refill() {
	now := time.Now()
	elapsed := now.Sub(tb.lastUpdate).Seconds()
	tb.tokens += elapsed * tb.rate
	if tb.tokens > tb.capacity {
		tb.tokens = tb.capacity
	}
	tb.lastUpdate = now
}

// Allow reports whether one token is available and consumes it.
func (tb *Local) Allow() bool {
	return tb.AllowN(1)
}

// AllowN reports whether n tokens are available and consumes them.
func (tb *Local) AllowN(n int) bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.refill()
	if tb.tokens >= float64(n) {
		tb.tokens -= float64(n)
		return true
	}
	return false
}

// Wait blocks until a token is available or ctx is cancelled.
func (tb *Local) Wait(ctx context.Context) error {
	for {
		if tb.Allow() {
			return nil
		}
		wait := tb.Reserve()
		if wait <= 0 {
			wait = 10 * time.Millisecond
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wait):
		}
	}
}

// Reserve returns how long to wait for the next token.
func (tb *Local) Reserve() time.Duration {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.refill()
	if tb.tokens >= 1 {
		return 0
	}
	if tb.rate <= 0 {
		return time.Hour
	}
	needed := 1 - tb.tokens
	return time.Duration(needed/tb.rate*1000) * time.Millisecond
}

// Tokens returns the current number of available tokens.
func (tb *Local) Tokens() float64 {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	tb.refill()
	return tb.tokens
}

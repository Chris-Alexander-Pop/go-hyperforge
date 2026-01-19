package tokenbucket

import (
	"context"
	"testing"
	"time"
)

func TestInMemoryLimiter(t *testing.T) {
	// Rate: 10 per second, Burst: 5
	l := NewInMemory(10.0, 5)

	ctx := context.Background()
	key := "user1"

	// Should allow 5 immediately (burst)
	for i := 0; i < 5; i++ {
		res, err := l.Allow(ctx, key, 0, 0) // Limit/Period ignored by InMemory implementation for simplicity??
		// Wait, looking at code:
		// pkg/algorithms/ratelimit/tokenbucket/tokenbucket.go:89: func (l *InMemoryLimiter) Allow(ctx context.Context, key string, limit int64, period time.Duration)
		// The InMemoryLimiter uses its own rate/burst fields, ignoring limit/period args in Allow??
		// Code check:
		// func (l *InMemoryLimiter) Allow(...) { ... tokens += elapsed * l.rate ... if tokens > l.burst ... }
		// Yes, it ignores the passed limit/period.
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if !res.Allowed {
			t.Errorf("Request %d should be allowed", i)
		}
	}

	// 6th request should fail
	res, _ := l.Allow(ctx, key, 0, 0)
	if res.Allowed {
		t.Error("Request 6 should be denied (burst exceeded)")
	}

	// Wait 100ms (should refill 1 token: 10/s * 0.1s = 1)
	time.Sleep(110 * time.Millisecond)
	res, _ = l.Allow(ctx, key, 0, 0)
	if !res.Allowed {
		t.Error("Request after wait should be allowed")
	}
}

package slidingwindow

import (
	"context"
	"strconv"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/algorithms/ratelimit"
	"github.com/chris-alexander-pop/system-design-library/pkg/cache"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
)

// Limiter implements a sliding window counter using Redis/Cache.
type Limiter struct {
	store cache.Cache
}

// New creates a new SlidingWindow limiter.
func New(store cache.Cache) *Limiter {
	return &Limiter{store: store}
}

func (l *Limiter) Allow(ctx context.Context, key string, limit int64, period time.Duration) (*ratelimit.Result, error) {
	// Simple Fixed Window approximation for v1, as per original code
	now := time.Now()

	// Optimized key generation: Avoid time.Format and repeated string concatenation
	// Allocates a single buffer and converts to string, avoiding intermediate allocations from Format
	buf := make([]byte, 0, len(key)+21) // 21 for separator + max int64 digits
	buf = append(buf, key...)
	buf = append(buf, ':')
	buf = strconv.AppendInt(buf, now.Truncate(period).Unix(), 10)
	windowKey := string(buf)

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

	return &ratelimit.Result{
		Allowed:   count <= limit,
		Remaining: remaining,
		Reset:     reset,
	}, nil
}

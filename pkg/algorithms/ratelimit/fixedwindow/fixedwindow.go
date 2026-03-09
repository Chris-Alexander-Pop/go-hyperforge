package fixedwindow

import (
	"context"
	"strconv"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/algorithms/ratelimit"
	"github.com/chris-alexander-pop/system-design-library/pkg/cache"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
)

// Limiter implements a simple time-bucketed counter.
type Limiter struct {
	store cache.Cache
}

// New creates a new FixedWindow limiter.
func New(store cache.Cache) *Limiter {
	return &Limiter{store: store}
}

func (l *Limiter) Allow(ctx context.Context, key string, limit int64, period time.Duration) (*ratelimit.Result, error) {
	window := time.Now().Truncate(period).Unix()
	// ⚡ Bolt: Use string concatenation instead of fmt.Sprintf for lower allocation overhead
	cacheKey := "rl:fixed:" + key + ":" + strconv.FormatInt(window, 10)

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

	return &ratelimit.Result{
		Allowed:   curr <= limit,
		Remaining: remaining,
		Reset:     time.Duration(resetSeconds) * time.Second,
	}, nil
}

package slidingwindow

import (
	"context"
	"strconv"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/algorithms/ratelimit"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/cache"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
)

// Limiter implements a true sliding-window counter using two adjacent fixed
// windows weighted by elapsed time in the current window (Redis-style sliding
// window counter). This approximates a continuous sliding window while using
// only cache Get/Incr/Set primitives.
type Limiter struct {
	store cache.Cache
}

// New creates a new SlidingWindow limiter.
func New(store cache.Cache) *Limiter {
	return &Limiter{store: store}
}

func windowKey(key string, windowStart int64) string {
	buf := make([]byte, 0, len(key)+24)
	buf = append(buf, key...)
	buf = append(buf, ':')
	buf = strconv.AppendInt(buf, windowStart, 10)
	return string(buf)
}

func (l *Limiter) loadCount(ctx context.Context, key string) (int64, error) {
	var n int64
	err := l.store.Get(ctx, key, &n)
	if err != nil {
		if cache.IsNotFound(err) {
			return 0, nil
		}
		return 0, err
	}
	return n, nil
}

func (l *Limiter) Allow(ctx context.Context, key string, limit int64, period time.Duration) (*ratelimit.Result, error) {
	if l.store == nil {
		return nil, errors.InvalidArgument("slidingwindow Limiter requires a non-nil cache store", nil)
	}
	if limit <= 0 || period <= 0 {
		return nil, errors.InvalidArgument("limit and period must be positive", nil)
	}

	now := time.Now()
	periodSecs := period.Seconds()
	windowStart := now.Truncate(period).Unix()
	prevStart := windowStart - int64(period/time.Second)
	if period < time.Second {
		prevStart = windowStart - 1
	}

	currKey := windowKey(key, windowStart)
	prevKey := windowKey(key, prevStart)

	curr, err := l.loadCount(ctx, currKey)
	if err != nil {
		return nil, errors.Wrap(err, "slidingwindow curr get")
	}
	prev, err := l.loadCount(ctx, prevKey)
	if err != nil {
		return nil, errors.Wrap(err, "slidingwindow prev get")
	}

	elapsed := now.Sub(now.Truncate(period)).Seconds()
	weight := 1.0 - (elapsed / periodSecs)
	if weight < 0 {
		weight = 0
	}
	if weight > 1 {
		weight = 1
	}

	estimate := float64(prev)*weight + float64(curr+1)
	reset := time.Until(now.Truncate(period).Add(period))

	if estimate > float64(limit) {
		remaining := limit - int64(float64(prev)*weight+float64(curr))
		if remaining < 0 {
			remaining = 0
		}
		return &ratelimit.Result{
			Allowed:   false,
			Remaining: remaining,
			Reset:     reset,
		}, nil
	}

	newCurr, err := l.store.Incr(ctx, currKey, 1)
	if err != nil {
		return nil, errors.Wrap(err, "slidingwindow incr")
	}
	if newCurr == 1 {
		_ = l.store.Set(ctx, currKey, newCurr, period*2)
	}

	remaining := limit - int64(estimate)
	if remaining < 0 {
		remaining = 0
	}
	return &ratelimit.Result{
		Allowed:   true,
		Remaining: remaining,
		Reset:     reset,
	}, nil
}

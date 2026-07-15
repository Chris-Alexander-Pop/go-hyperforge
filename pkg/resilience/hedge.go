package resilience

import (
	"context"
	"time"
)

// Hedge runs fn immediately and, if it has not completed after delay, starts a
// speculative second attempt. The first successful result wins; remaining work
// is cancelled via context. If both fail, the first error is returned.
//
// delay <= 0 means no hedge (fn runs once). If the primary fails before delay
// elapses, a second attempt starts immediately.
func Hedge(ctx context.Context, delay time.Duration, fn Executor) error {
	_, err := HedgeT(ctx, delay, func(ctx context.Context) (struct{}, error) {
		return struct{}{}, fn(ctx)
	})
	return err
}

// HedgeT is the typed form of Hedge.
func HedgeT[T any](ctx context.Context, delay time.Duration, fn func(context.Context) (T, error)) (T, error) {
	var zero T
	if fn == nil {
		return zero, nil
	}
	if delay <= 0 {
		return fn(ctx)
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	type outcome struct {
		val T
		err error
	}
	results := make(chan outcome, 2)

	start := func() {
		v, err := fn(ctx)
		results <- outcome{val: v, err: err}
	}

	go start()

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case o := <-results:
		if o.err == nil {
			return o.val, nil
		}
		// Primary failed before hedge delay — try once more immediately.
		go start()
		select {
		case o2 := <-results:
			if o2.err == nil {
				return o2.val, nil
			}
			return zero, o.err
		case <-ctx.Done():
			return zero, o.err
		}

	case <-timer.C:
		go start()
		var firstErr error
		for i := 0; i < 2; i++ {
			select {
			case o := <-results:
				if o.err == nil {
					cancel()
					return o.val, nil
				}
				if firstErr == nil {
					firstErr = o.err
				}
			case <-ctx.Done():
				if firstErr != nil {
					return zero, firstErr
				}
				return zero, ctx.Err()
			}
		}
		return zero, firstErr

	case <-ctx.Done():
		return zero, ctx.Err()
	}
}

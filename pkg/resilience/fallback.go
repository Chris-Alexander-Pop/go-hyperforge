package resilience

import "context"

// Fallback runs primary; if it returns a non-nil error, runs secondary and
// returns secondary's result. Context cancellation on primary still triggers
// fallback unless the caller wants otherwise (check ctx before calling).
func Fallback(ctx context.Context, primary, secondary Executor) error {
	if primary == nil {
		if secondary == nil {
			return nil
		}
		return secondary(ctx)
	}
	if err := primary(ctx); err != nil {
		if secondary == nil {
			return err
		}
		return secondary(ctx)
	}
	return nil
}

// FallbackT is the typed form of Fallback.
func FallbackT[T any](ctx context.Context, primary, secondary func(context.Context) (T, error)) (T, error) {
	var zero T
	if primary == nil {
		if secondary == nil {
			return zero, nil
		}
		return secondary(ctx)
	}
	val, err := primary(ctx)
	if err != nil {
		if secondary == nil {
			return zero, err
		}
		return secondary(ctx)
	}
	return val, nil
}

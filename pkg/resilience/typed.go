package resilience

import "context"

// ExecutorT is a typed unit of work returning (T, error).
type ExecutorT[T any] func(ctx context.Context) (T, error)

// ExecuteT runs a typed function under a Breaker, capturing the successful value.
func ExecuteT[T any](ctx context.Context, b Breaker, fn ExecutorT[T]) (T, error) {
	var zero T
	if b == nil {
		if fn == nil {
			return zero, nil
		}
		return fn(ctx)
	}
	var result T
	err := b.Execute(ctx, func(ctx context.Context) error {
		var err error
		result, err = fn(ctx)
		return err
	})
	if err != nil {
		return zero, err
	}
	return result, nil
}

// RetryT runs a typed function with the given retry policy.
func RetryT[T any](ctx context.Context, cfg RetryConfig, fn ExecutorT[T]) (T, error) {
	var result T
	err := Retry(ctx, cfg, func(ctx context.Context) error {
		var err error
		result, err = fn(ctx)
		return err
	})
	if err != nil {
		var zero T
		return zero, err
	}
	return result, nil
}

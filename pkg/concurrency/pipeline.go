package concurrency

import (
	"context"
	"sync"
)

// Generator creates a channel that yields values from a slice.
func Generator[T any](ctx context.Context, items ...T) <-chan T {
	out := make(chan T)
	go func() {
		defer close(out)
		for _, item := range items {
			select {
			case <-ctx.Done():
				return
			case out <- item:
			}
		}
	}()
	return out
}

// Stage represents a pipeline stage that transforms input to output.
type Stage[In, Out any] func(ctx context.Context, in In) (Out, error)

// Pipeline chains multiple processing stages together.
// Each stage runs in its own goroutine.
func Pipeline[In, Out any](
	ctx context.Context,
	input <-chan In,
	process func(context.Context, In) (Out, error),
) <-chan Out {
	out := make(chan Out)
	go func() {
		defer close(out)
		for in := range input {
			select {
			case <-ctx.Done():
				return
			default:
			}

			result, err := process(ctx, in)
			if err != nil {
				continue // Skip errors, or could send to error channel
			}

			select {
			case <-ctx.Done():
				return
			case out <- result:
			}
		}
	}()
	return out
}

// PipelineWithErrors returns both results and errors.
type Result[T any] struct {
	Value T
	Err   error
}

func PipelineWithErrors[In, Out any](
	ctx context.Context,
	input <-chan In,
	process func(context.Context, In) (Out, error),
) <-chan Result[Out] {
	out := make(chan Result[Out])
	go func() {
		defer close(out)
		for in := range input {
			select {
			case <-ctx.Done():
				return
			default:
			}

			result, err := process(ctx, in)

			select {
			case <-ctx.Done():
				return
			case out <- Result[Out]{Value: result, Err: err}:
			}
		}
	}()
	return out
}

// FanOutFanIn distributes work across multiple workers and collects results.
func FanOutFanIn[In, Out any](
	ctx context.Context,
	input <-chan In,
	workers int,
	process func(context.Context, In) (Out, error),
) <-chan Out {
	// Fan-out: create multiple worker channels
	channels := make([]<-chan Out, workers)
	for i := 0; i < workers; i++ {
		channels[i] = Pipeline(ctx, input, process)
	}

	// Fan-in: merge all worker outputs
	return FanIn(ctx, channels...)
}

// FanIn merges multiple channels into one.
func FanIn[T any](ctx context.Context, channels ...<-chan T) <-chan T {
	out := make(chan T)
	var wg sync.WaitGroup

	output := func(ch <-chan T) {
		defer wg.Done()
		for val := range ch {
			select {
			case <-ctx.Done():
				return
			case out <- val:
			}
		}
	}

	wg.Add(len(channels))
	for _, ch := range channels {
		go output(ch)
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}

// Batch collects items into fixed-size batches.
func Batch[T any](ctx context.Context, input <-chan T, size int) <-chan []T {
	out := make(chan []T)
	go func() {
		defer close(out)
		batch := make([]T, 0, size)

		for item := range input {
			batch = append(batch, item)
			if len(batch) >= size {
				select {
				case <-ctx.Done():
					return
				case out <- batch:
				}
				batch = make([]T, 0, size)
			}
		}

		// Send remaining items
		if len(batch) > 0 {
			select {
			case <-ctx.Done():
				return
			case out <- batch:
			}
		}
	}()
	return out
}

// Filter keeps only items that match the predicate.
func Filter[T any](ctx context.Context, input <-chan T, predicate func(T) bool) <-chan T {
	out := make(chan T)
	go func() {
		defer close(out)
		for item := range input {
			if predicate(item) {
				select {
				case <-ctx.Done():
					return
				case out <- item:
				}
			}
		}
	}()
	return out
}

// Map transforms each item using the given function.
func Map[In, Out any](ctx context.Context, input <-chan In, fn func(In) Out) <-chan Out {
	out := make(chan Out)
	go func() {
		defer close(out)
		for item := range input {
			select {
			case <-ctx.Done():
				return
			case out <- fn(item):
			}
		}
	}()
	return out
}

// Take returns the first n items.
func Take[T any](ctx context.Context, input <-chan T, n int) <-chan T {
	out := make(chan T)
	go func() {
		defer close(out)
		count := 0
		for item := range input {
			if count >= n {
				return
			}
			select {
			case <-ctx.Done():
				return
			case out <- item:
				count++
			}
		}
	}()
	return out
}

// OrDone wraps a channel to respect context cancellation.
func OrDone[T any](ctx context.Context, input <-chan T) <-chan T {
	out := make(chan T)
	go func() {
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				return
			case val, ok := <-input:
				if !ok {
					return
				}
				select {
				case out <- val:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return out
}

// Tee duplicates a channel's output to two channels.
func Tee[T any](ctx context.Context, input <-chan T) (<-chan T, <-chan T) {
	out1 := make(chan T)
	out2 := make(chan T)

	go func() {
		defer close(out1)
		defer close(out2)

		for val := range OrDone(ctx, input) {
			// Create local copies for each select
			var o1, o2 = out1, out2
			for i := 0; i < 2; i++ {
				select {
				case o1 <- val:
					o1 = nil // Disable this case after sending
				case o2 <- val:
					o2 = nil
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return out1, out2
}

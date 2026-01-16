package concurrency

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"

	"github.com/chris-alexander-pop/system-design-library/pkg/logger"
)

// SafeGo runs the function in a goroutine and recovers from panics
func SafeGo(ctx context.Context, fn func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				err := fmt.Errorf("panic recovered: %v", r)
				stack := string(debug.Stack())
				logger.L().ErrorContext(ctx, "goroutine panic", "error", err, "stack", stack)
			}
		}()
		fn()
	}()
}

// FanOut runs 'n' copies of the task concurrently and waits for all to finish
func FanOut(ctx context.Context, n int, fn func(i int)) {
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		idx := i
		SafeGo(ctx, func() {
			defer wg.Done()
			fn(idx)
		})
	}
	wg.Wait()
}

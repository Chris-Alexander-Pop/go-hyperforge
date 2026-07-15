package concurrency

import (
	"context"
	"sync"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/algorithms/concurrency/adaptive"
)

type Task func(ctx context.Context)

// WorkerPoolOption configures optional WorkerPool behavior.
type WorkerPoolOption func(*WorkerPool)

// WithAdaptiveLimiter wires pkg/algorithms/concurrency/adaptive into the pool.
// Submit still queues work; workers acquire a token before running each task and
// release with measured latency so the limit adapts. When Acquire fails, the
// worker waits briefly and retries until the context is canceled or a token is
// granted (load shedding via delayed execution rather than drop).
func WithAdaptiveLimiter(minLimit, maxLimit float64) WorkerPoolOption {
	return func(wp *WorkerPool) {
		wp.adaptive = adaptive.New(minLimit, maxLimit)
	}
}

type WorkerPool struct {
	maxWorkers int
	taskQueue  chan Task
	wg         sync.WaitGroup
	adaptive   *adaptive.Limiter
}

func NewWorkerPool(maxWorkers int, queueSize int, opts ...WorkerPoolOption) *WorkerPool {
	wp := &WorkerPool{
		maxWorkers: maxWorkers,
		taskQueue:  make(chan Task, queueSize),
	}
	for _, opt := range opts {
		opt(wp)
	}
	return wp
}

func (wp *WorkerPool) Start(ctx context.Context) {
	for i := 0; i < wp.maxWorkers; i++ {
		wp.wg.Add(1)
		go wp.worker(ctx)
	}
}

func (wp *WorkerPool) worker(ctx context.Context) {
	defer wp.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case task, ok := <-wp.taskQueue:
			if !ok {
				return
			}
			wp.runTask(ctx, task)
		}
	}
}

func (wp *WorkerPool) runTask(ctx context.Context, task Task) {
	if wp.adaptive != nil {
		for !wp.adaptive.Acquire() {
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Millisecond):
			}
		}
		start := time.Now()
		task(ctx)
		wp.adaptive.Release(time.Since(start))
		return
	}
	task(ctx)
}

// Submit adds a task. Blocks if the queue is full.
func (wp *WorkerPool) Submit(task Task) {
	wp.taskQueue <- task
}

// AdaptiveLimit returns the current adaptive concurrency limit, or 0 if unused.
func (wp *WorkerPool) AdaptiveLimit() float64 {
	if wp.adaptive == nil {
		return 0
	}
	return wp.adaptive.Limit()
}

// Stop waits for workers to finish
func (wp *WorkerPool) Stop() {
	close(wp.taskQueue)
	wp.wg.Wait()
}

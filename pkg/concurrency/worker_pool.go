package concurrency

import (
	"context"
	"sync"
)

type Task func(ctx context.Context)

type WorkerPool struct {
	maxWorkers int
	taskQueue  chan Task
	wg         sync.WaitGroup
}

func NewWorkerPool(maxWorkers int, queueSize int) *WorkerPool {
	return &WorkerPool{
		maxWorkers: maxWorkers,
		taskQueue:  make(chan Task, queueSize),
	}
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
			task(ctx)
		}
	}
}

// Submit adds a task. Returns false if queue is full (non-blocking) or blocks?
// Overengineered: Option to Block or Drop.
// Simplest robust: Block.
func (wp *WorkerPool) Submit(task Task) {
	wp.taskQueue <- task
}

// Stop waits for workers to finish
func (wp *WorkerPool) Stop() {
	close(wp.taskQueue)
	wp.wg.Wait()
}

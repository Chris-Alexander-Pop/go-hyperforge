/*
Package concurrency provides advanced concurrency primitives with observability.

Features:
  - SmartMutex / SmartRWMutex: Deadlock detection and slow lock logging
  - Semaphore: Weighted semaphore (local implementation)
  - NewWeighted / Weighted: re-export of golang.org/x/sync/semaphore
  - ErrGroup / ErrGroupWithContext: re-export of golang.org/x/sync/errgroup
  - WorkerPool: Goroutine pool (optional WithAdaptiveLimiter)
  - Group: singleflight-style request coalescing
  - Pipeline: Data processing pipeline
*/
package concurrency

/*
Package concurrency provides advanced concurrency primitives with observability.

Features:
  - SmartMutex / SmartRWMutex: Deadlock detection and slow lock logging
  - Semaphore: Weighted semaphore
  - WorkerPool: Goroutine pool (optional WithAdaptiveLimiter)
  - Group: singleflight-style request coalescing
  - Pipeline: Data processing pipeline
*/
package concurrency

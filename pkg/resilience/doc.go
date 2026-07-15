/*
Package resilience provides common patterns for building robust, fault-tolerant systems.

This package implements:
  - Circuit Breaker: Prevents cascading failures by stopping requests to failing services.
  - Retry: Automatically retries failed operations with exponential backoff and jitter.
  - Timeout: Deadline enforcement via WithTimeout.
  - Bulkhead: Semaphore-bounded concurrency isolation via pkg/concurrency.

Prefer this package over mesh-facing facades in pkg/servicemesh for application code.
Observability belongs on InstrumentedCircuitBreaker; NewCircuitBreaker stays quiet.

Usage:

	import "github.com/chris-alexander-pop/system-design-library/pkg/resilience"

	// Circuit Breaker
	cb := resilience.NewCircuitBreaker(resilience.DefaultCircuitBreakerConfig("my-service"))

	err := cb.Execute(ctx, func(ctx context.Context) error {
	    return upstream.Call(ctx)
	})

	// Instrumented (logging + tracing)
	icb := resilience.NewInstrumentedBreakerFromConfig(resilience.DefaultCircuitBreakerConfig("my-service"))

	// Retry
	err := resilience.Retry(ctx, resilience.DefaultRetryConfig(), func(ctx context.Context) error {
	    return upstream.Call(ctx)
	})

	// Bulkhead
	bh := resilience.NewBulkhead(resilience.BulkheadConfig{Name: "db", MaxConcurrent: 16})
	err = bh.Execute(ctx, func(ctx context.Context) error {
	    return db.Query(ctx)
	})
*/
package resilience

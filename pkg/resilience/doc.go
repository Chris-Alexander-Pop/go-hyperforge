/*
Package resilience provides common patterns for building robust, fault-tolerant systems.

This package implements:
  - Circuit Breaker: Prevents cascading failures by stopping requests to failing services.
    Half-open probes are capped by MaxRequests (ErrTooManyRequests when exceeded).
  - Retry: Automatically retries failed operations with exponential backoff and jitter
    (package-level Retry helper and Retrier interface via NewRetrier).
  - Timeout: Deadline enforcement via WithTimeout (returns CodeDeadlineExceeded).
  - Bulkhead: Semaphore-bounded concurrency isolation via pkg/concurrency.
  - Hedge: Speculative retry after a delay (first success wins).
  - Fallback: Primary-then-secondary execution helpers.
  - Typed Execute: ExecuteT / RetryT / HedgeT / FallbackT returning (T, error).

Env-tagged Config (RESILIENCE_*) derives CircuitBreaker / Retry / Bulkhead configs.

Error mapping (pkg/errors):
  - ErrCircuitOpen → UNAVAILABLE (HTTP 503)
  - ErrTooManyRequests / ErrBulkheadFull → RESOURCE_EXHAUSTED (HTTP 429)

Prefer this package over mesh-facing facades in pkg/servicemesh for application code.
Observability belongs on InstrumentedCircuitBreaker; NewCircuitBreaker stays quiet.

Usage:

	import "github.com/chris-alexander-pop/system-design-library/pkg/resilience"

	// Circuit Breaker
	cb := resilience.NewCircuitBreaker(resilience.DefaultCircuitBreakerConfig("my-service"))

	err := cb.Execute(ctx, func(ctx context.Context) error {
	    return upstream.Call(ctx)
	})

	// Typed execute
	val, err := resilience.ExecuteT(ctx, cb, func(ctx context.Context) (string, error) {
	    return upstream.Fetch(ctx)
	})

	// Hedge + Fallback
	err = resilience.Hedge(ctx, 50*time.Millisecond, upstream.Call)
	err = resilience.Fallback(ctx, primary.Call, secondary.Call)

	// Env-tagged config
	cfg := resilience.DefaultConfig()
	cb = resilience.NewCircuitBreaker(cfg.CircuitBreaker())

	// Instrumented (logging + tracing)
	icb := resilience.NewInstrumentedBreakerFromConfig(resilience.DefaultCircuitBreakerConfig("my-service"))

	// Retry (helper or Retrier)
	err := resilience.Retry(ctx, resilience.DefaultRetryConfig(), func(ctx context.Context) error {
	    return upstream.Call(ctx)
	})
	retrier := resilience.NewRetrier(resilience.DefaultRetryConfig())
	err = retrier.Execute(ctx, upstream.Call)

	// Bulkhead
	bh := resilience.NewBulkhead(resilience.BulkheadConfig{Name: "db", MaxConcurrent: 16})
	err = bh.Execute(ctx, func(ctx context.Context) error {
	    return db.Query(ctx)
	})
*/
package resilience

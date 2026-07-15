package resilience

import (
	"context"
	"time"
)

// State represents the current state of a circuit breaker.
type State string

const (
	StateClosed   State = "closed"    // Normal operation, tracking failures
	StateOpen     State = "open"      // Blocking requests, fast-fail
	StateHalfOpen State = "half_open" // Testing if service has recovered
)

// Executor is a unit of work that can run under resilience patterns.
type Executor func(ctx context.Context) error

// Breaker is the thin circuit-breaker interface for reuse and decoration.
// The concrete *CircuitBreaker and *InstrumentedCircuitBreaker implement it.
type Breaker interface {
	Execute(ctx context.Context, fn Executor) error
	State() State
	Reset()
	Metrics() CircuitBreakerMetrics
}

// Retrier is the thin retry interface for reuse and decoration.
// Prefer NewRetrier when callers need a reusable policy object; Retry remains
// available as a package-level helper.
type Retrier interface {
	Execute(ctx context.Context, fn Executor) error
}

// RetryPolicy is a concrete Retrier backed by RetryConfig.
type RetryPolicy struct {
	cfg RetryConfig
}

// NewRetrier creates a Retrier from the given retry configuration.
func NewRetrier(cfg RetryConfig) *RetryPolicy {
	return &RetryPolicy{cfg: cfg}
}

// Execute runs fn with the configured retry policy.
func (r *RetryPolicy) Execute(ctx context.Context, fn Executor) error {
	return Retry(ctx, r.cfg, fn)
}

var _ Retrier = (*RetryPolicy)(nil)

// CircuitBreakerConfig configures the circuit breaker behavior.
type CircuitBreakerConfig struct {
	// Name identifies this circuit breaker (for logging/metrics).
	Name string

	// FailureThreshold is the number of failures before opening the circuit.
	FailureThreshold int64

	// SuccessThreshold is the number of successes in half-open state to close.
	SuccessThreshold int64

	// Timeout is how long to wait before transitioning from open to half-open.
	Timeout time.Duration

	// MaxRequests is the max concurrent/allowed probes in half-open state.
	// Zero means unlimited (backward compatible).
	MaxRequests int64

	// OnStateChange is called when the circuit breaker changes state.
	OnStateChange func(name string, from, to State)
}

// RetryConfig configures retry behavior.
type RetryConfig struct {
	// MaxAttempts is the maximum number of attempts (including the first).
	MaxAttempts int

	// InitialBackoff is the backoff duration for the first retry.
	InitialBackoff time.Duration

	// MaxBackoff caps the backoff duration.
	MaxBackoff time.Duration

	// Multiplier increases the backoff between retries.
	Multiplier float64

	// Jitter adds randomness to prevent thundering herd.
	Jitter float64

	// RetryIf determines if an error should be retried.
	RetryIf func(error) bool
}

// DefaultCircuitBreakerConfig returns sensible defaults.
func DefaultCircuitBreakerConfig(name string) CircuitBreakerConfig {
	return CircuitBreakerConfig{
		Name:             name,
		FailureThreshold: 5,
		SuccessThreshold: 2,
		Timeout:          30 * time.Second,
		MaxRequests:      0,
	}
}

// DefaultRetryConfig returns sensible defaults.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:    3,
		InitialBackoff: 100 * time.Millisecond,
		MaxBackoff:     10 * time.Second,
		Multiplier:     2.0,
		Jitter:         0.1,
		RetryIf:        func(err error) bool { return err != nil },
	}
}

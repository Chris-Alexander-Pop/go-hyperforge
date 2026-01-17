// Package resilience provides patterns for building resilient systems.
//
// This package includes:
//   - Circuit Breaker: Prevents cascading failures
//   - Retry: Automatic retries with backoff
//   - Timeout: Request deadline enforcement
//   - Bulkhead: Isolation of resources
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

	// OnStateChange is called when the circuit breaker changes state.
	OnStateChange func(name string, from, to State)
}

// Executor represents something that can be executed with circuit breaker protection.
type Executor func(ctx context.Context) error

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

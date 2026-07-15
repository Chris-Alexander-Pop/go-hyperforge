// Package circuitbreaker is a mesh-facing facade over pkg/resilience.
//
// Prefer github.com/chris-alexander-pop/system-design-library/pkg/resilience
// for application code. This package keeps the historical Options/Execute
// shapes used by service-mesh integrations while delegating state and
// execution to resilience.CircuitBreaker.
//
// Usage:
//
//	cb := circuitbreaker.New("payment-service", circuitbreaker.Options{
//	    FailureThreshold: 5,
//	    Timeout: 30 * time.Second,
//	})
//	result, err := cb.Execute(func() (interface{}, error) {
//	    return client.ProcessPayment(ctx, payment)
//	})
package circuitbreaker

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"github.com/chris-alexander-pop/system-design-library/pkg/resilience"
)

// State represents the circuit breaker state (mesh-facing string values).
type State string

const (
	StateClosed   State = "closed"
	StateOpen     State = "open"
	StateHalfOpen State = "half-open"
)

// Options configures the circuit breaker.
type Options struct {
	// FailureThreshold is consecutive failures before opening.
	FailureThreshold int

	// SuccessThreshold is successes needed to close from half-open.
	SuccessThreshold int

	// Timeout is duration to stay open before half-open.
	Timeout time.Duration

	// MaxRequests is max requests allowed in half-open state.
	MaxRequests int

	// OnStateChange is called when state changes.
	OnStateChange func(from, to State)
}

// CircuitBreaker is a thin wrapper around resilience.CircuitBreaker.
type CircuitBreaker struct {
	name  string
	opts  Options
	inner *resilience.CircuitBreaker
}

// New creates a new circuit breaker that delegates to pkg/resilience.
func New(name string, opts Options) *CircuitBreaker {
	if opts.FailureThreshold <= 0 {
		opts.FailureThreshold = 5
	}
	if opts.SuccessThreshold <= 0 {
		opts.SuccessThreshold = 2
	}
	if opts.Timeout <= 0 {
		opts.Timeout = 30 * time.Second
	}
	if opts.MaxRequests <= 0 {
		opts.MaxRequests = 1
	}

	cb := &CircuitBreaker{name: name, opts: opts}

	cfg := resilience.CircuitBreakerConfig{
		Name:             name,
		FailureThreshold: int64(opts.FailureThreshold),
		SuccessThreshold: int64(opts.SuccessThreshold),
		Timeout:          opts.Timeout,
		MaxRequests:      int64(opts.MaxRequests),
		OnStateChange: func(_ string, from, to resilience.State) {
			if opts.OnStateChange != nil {
				go opts.OnStateChange(mapState(from), mapState(to))
			}
		},
	}
	cb.inner = resilience.NewCircuitBreaker(cfg)
	return cb
}

// Execute runs the function with circuit breaker protection.
func (cb *CircuitBreaker) Execute(fn func() (interface{}, error)) (interface{}, error) {
	var result interface{}
	err := cb.inner.Execute(context.Background(), func(ctx context.Context) error {
		var execErr error
		result, execErr = fn()
		return execErr
	})
	return result, mapError(err)
}

// ExecuteContext runs the function with context and circuit breaker protection.
func (cb *CircuitBreaker) ExecuteContext(ctx context.Context, fn func(context.Context) (interface{}, error)) (interface{}, error) {
	var result interface{}
	err := cb.inner.Execute(ctx, func(ctx context.Context) error {
		var execErr error
		result, execErr = fn(ctx)
		return execErr
	})
	return result, mapError(err)
}

// State returns the current state.
func (cb *CircuitBreaker) State() State {
	return mapState(cb.inner.State())
}

// Name returns the circuit breaker name.
func (cb *CircuitBreaker) Name() string {
	return cb.name
}

// Metrics returns current metrics.
func (cb *CircuitBreaker) Metrics() Metrics {
	m := cb.inner.Metrics()
	return Metrics{
		State:       mapState(m.State),
		Failures:    int(m.Failures),
		Successes:   int(m.Successes),
		LastFailure: m.LastFailure,
	}
}

// ForceOpen forces the circuit to open state.
func (cb *CircuitBreaker) ForceOpen() {
	cb.inner.ForceOpen()
}

// ForceClose forces the circuit to closed state.
func (cb *CircuitBreaker) ForceClose() {
	cb.inner.ForceClose()
}

// Metrics contains circuit breaker metrics.
type Metrics struct {
	State       State
	Failures    int
	Successes   int
	LastFailure time.Time
}

func mapState(s resilience.State) State {
	switch s {
	case resilience.StateClosed:
		return StateClosed
	case resilience.StateOpen:
		return StateOpen
	case resilience.StateHalfOpen:
		return StateHalfOpen
	default:
		return State(s)
	}
}

func mapError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, resilience.ErrCircuitOpen) {
		// Half-open max-request rejection also surfaces as circuit-open from
		// resilience; preserve historical ErrTooManyRequests when already half-open
		// is not distinguishable here, so mesh callers get ErrCircuitOpen which
		// matches open-circuit rejection tests. ErrTooManyRequests remains exported
		// for callers that probe state themselves.
		return ErrCircuitOpen
	}
	return err
}

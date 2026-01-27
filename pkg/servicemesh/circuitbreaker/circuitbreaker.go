// Package circuitbreaker provides circuit breaker pattern implementation.
//
// The circuit breaker prevents cascading failures by temporarily blocking
// requests to failing services. It supports three states:
//   - Closed: Requests flow normally
//   - Open: Requests fail immediately
//   - Half-Open: Limited requests allowed to test recovery
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
	"sync"
	"time"
)

// State represents the circuit breaker state.
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

// CircuitBreaker implements the circuit breaker pattern.
type CircuitBreaker struct {
	name    string
	options Options

	mu            sync.RWMutex
	state         State
	failures      int
	successes     int
	lastFailure   time.Time
	halfOpenCount int
}

// New creates a new circuit breaker.
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

	return &CircuitBreaker{
		name:    name,
		options: opts,
		state:   StateClosed,
	}
}

// Execute runs the function with circuit breaker protection.
func (cb *CircuitBreaker) Execute(fn func() (interface{}, error)) (interface{}, error) {
	if err := cb.beforeRequest(); err != nil {
		return nil, err
	}

	result, err := fn()

	cb.afterRequest(err == nil)

	return result, err
}

// ExecuteContext runs the function with context and circuit breaker protection.
func (cb *CircuitBreaker) ExecuteContext(ctx context.Context, fn func(context.Context) (interface{}, error)) (interface{}, error) {
	if err := cb.beforeRequest(); err != nil {
		return nil, err
	}

	result, err := fn(ctx)

	cb.afterRequest(err == nil)

	return result, err
}

func (cb *CircuitBreaker) beforeRequest() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		return nil

	case StateOpen:
		// Check if timeout has passed
		if time.Since(cb.lastFailure) > cb.options.Timeout {
			cb.setState(StateHalfOpen)
			cb.halfOpenCount = 1
			return nil
		}
		return ErrCircuitOpen

	case StateHalfOpen:
		if cb.halfOpenCount >= cb.options.MaxRequests {
			return ErrTooManyRequests
		}
		cb.halfOpenCount++
		return nil
	}

	return nil
}

func (cb *CircuitBreaker) afterRequest(success bool) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		if success {
			cb.failures = 0
		} else {
			cb.failures++
			cb.lastFailure = time.Now()
			if cb.failures >= cb.options.FailureThreshold {
				cb.setState(StateOpen)
			}
		}

	case StateHalfOpen:
		if success {
			cb.successes++
			if cb.successes >= cb.options.SuccessThreshold {
				cb.setState(StateClosed)
			}
		} else {
			cb.setState(StateOpen)
		}
	}
}

func (cb *CircuitBreaker) setState(state State) {
	if cb.state == state {
		return
	}

	from := cb.state
	cb.state = state

	// Reset counters
	cb.failures = 0
	cb.successes = 0
	cb.halfOpenCount = 0

	if state == StateOpen {
		cb.lastFailure = time.Now()
	}

	if cb.options.OnStateChange != nil {
		go cb.options.OnStateChange(from, state)
	}
}

// State returns the current state.
func (cb *CircuitBreaker) State() State {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Name returns the circuit breaker name.
func (cb *CircuitBreaker) Name() string {
	return cb.name
}

// Metrics returns current metrics.
func (cb *CircuitBreaker) Metrics() Metrics {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return Metrics{
		State:       cb.state,
		Failures:    cb.failures,
		Successes:   cb.successes,
		LastFailure: cb.lastFailure,
	}
}

// ForceOpen forces the circuit to open state.
func (cb *CircuitBreaker) ForceOpen() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.setState(StateOpen)
}

// ForceClose forces the circuit to closed state.
func (cb *CircuitBreaker) ForceClose() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.setState(StateClosed)
}

// Metrics contains circuit breaker metrics.
type Metrics struct {
	State       State
	Failures    int
	Successes   int
	LastFailure time.Time
}

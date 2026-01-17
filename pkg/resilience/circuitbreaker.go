package resilience

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"github.com/chris-alexander-pop/system-design-library/pkg/logger"
)

// Error codes for circuit breaker
const (
	CodeCircuitOpen = "CIRCUIT_OPEN"
)

// ErrCircuitOpen is returned when the circuit is open.
var ErrCircuitOpen = errors.New(CodeCircuitOpen, "circuit breaker is open", nil)

// CircuitBreaker implements the circuit breaker pattern.
//
// States:
//   - Closed: Normal operation. Failures are counted.
//   - Open: All requests fail fast. After timeout, transitions to half-open.
//   - Half-Open: Limited requests are allowed to test recovery.
type CircuitBreaker struct {
	config CircuitBreakerConfig

	state       atomic.Value // State
	failures    atomic.Int64
	successes   atomic.Int64
	lastFailure atomic.Int64 // Unix timestamp
	mu          sync.RWMutex
}

// NewCircuitBreaker creates a new circuit breaker.
func NewCircuitBreaker(cfg CircuitBreakerConfig) *CircuitBreaker {
	if cfg.FailureThreshold == 0 {
		cfg.FailureThreshold = 5
	}
	if cfg.SuccessThreshold == 0 {
		cfg.SuccessThreshold = 2
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}

	cb := &CircuitBreaker{
		config: cfg,
	}
	cb.state.Store(StateClosed)
	return cb
}

// Execute runs the given function with circuit breaker protection.
func (cb *CircuitBreaker) Execute(ctx context.Context, fn Executor) error {
	// Check if we should allow the request
	if !cb.allowRequest() {
		return ErrCircuitOpen
	}

	// Execute
	err := fn(ctx)

	// Record result
	if err != nil {
		cb.recordFailure()
	} else {
		cb.recordSuccess()
	}

	return err
}

// State returns the current state of the circuit breaker.
func (cb *CircuitBreaker) State() State {
	return cb.state.Load().(State)
}

// Reset manually resets the circuit breaker to closed state.
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.setState(StateClosed)
	cb.failures.Store(0)
	cb.successes.Store(0)
}

func (cb *CircuitBreaker) allowRequest() bool {
	state := cb.State()

	switch state {
	case StateClosed:
		return true

	case StateOpen:
		// Check if timeout has passed
		lastFailure := time.UnixMilli(cb.lastFailure.Load())
		if time.Since(lastFailure) > cb.config.Timeout {
			cb.mu.Lock()
			// Double-check under lock
			if cb.State() == StateOpen {
				cb.setState(StateHalfOpen)
				cb.successes.Store(0)
				logger.L().Info("circuit breaker transitioning to half-open",
					"name", cb.config.Name)
			}
			cb.mu.Unlock()
			return true
		}
		return false

	case StateHalfOpen:
		return true
	}

	return false
}

func (cb *CircuitBreaker) recordSuccess() {
	state := cb.State()

	switch state {
	case StateClosed:
		// Reset failure count on success
		cb.failures.Store(0)

	case StateHalfOpen:
		successes := cb.successes.Add(1)
		if successes >= cb.config.SuccessThreshold {
			cb.mu.Lock()
			if cb.State() == StateHalfOpen {
				cb.setState(StateClosed)
				cb.failures.Store(0)
				logger.L().Info("circuit breaker closed",
					"name", cb.config.Name,
					"successes", successes)
			}
			cb.mu.Unlock()
		}
	}
}

func (cb *CircuitBreaker) recordFailure() {
	state := cb.State()
	cb.lastFailure.Store(time.Now().UnixMilli())

	switch state {
	case StateClosed:
		failures := cb.failures.Add(1)
		if failures >= cb.config.FailureThreshold {
			cb.mu.Lock()
			if cb.State() == StateClosed {
				cb.setState(StateOpen)
				logger.L().Warn("circuit breaker opened",
					"name", cb.config.Name,
					"failures", failures)
			}
			cb.mu.Unlock()
		}

	case StateHalfOpen:
		// Any failure in half-open goes back to open
		cb.mu.Lock()
		if cb.State() == StateHalfOpen {
			cb.setState(StateOpen)
			logger.L().Warn("circuit breaker reopened from half-open",
				"name", cb.config.Name)
		}
		cb.mu.Unlock()
	}
}

func (cb *CircuitBreaker) setState(newState State) {
	oldState := cb.State()
	if oldState != newState {
		cb.state.Store(newState)
		if cb.config.OnStateChange != nil {
			cb.config.OnStateChange(cb.config.Name, oldState, newState)
		}
	}
}

// Metrics returns current circuit breaker metrics.
func (cb *CircuitBreaker) Metrics() CircuitBreakerMetrics {
	return CircuitBreakerMetrics{
		State:       cb.State(),
		Failures:    cb.failures.Load(),
		Successes:   cb.successes.Load(),
		LastFailure: time.UnixMilli(cb.lastFailure.Load()),
	}
}

// CircuitBreakerMetrics contains circuit breaker statistics.
type CircuitBreakerMetrics struct {
	State       State
	Failures    int64
	Successes   int64
	LastFailure time.Time
}

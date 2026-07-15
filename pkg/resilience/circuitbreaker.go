package resilience

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/concurrency"
)

// CircuitBreaker implements the circuit breaker pattern.
//
// States:
//   - Closed: Normal operation. Failures are counted.
//   - Open: All requests fail fast. After timeout, transitions to half-open.
//   - Half-Open: Limited requests are allowed to test recovery.
//
// Logging and tracing belong on InstrumentedCircuitBreaker; this type stays quiet.
type CircuitBreaker struct {
	config CircuitBreakerConfig

	state         atomic.Value // State
	failures      atomic.Int64
	successes     atomic.Int64
	lastFailure   atomic.Int64 // Unix timestamp
	halfOpenCount atomic.Int64
	mu            *concurrency.SmartRWMutex
}

// NewCircuitBreaker creates a new circuit breaker (concrete, uninstrumented).
// Use NewInstrumentedBreakerFromConfig when observability is desired.
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
		mu:     concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "CircuitBreaker-" + cfg.Name}),
	}
	cb.state.Store(StateClosed)
	return cb
}

// Execute runs the given function with circuit breaker protection.
func (cb *CircuitBreaker) Execute(ctx context.Context, fn Executor) error {
	if !cb.allowRequest() {
		return ErrCircuitOpen
	}

	err := fn(ctx)

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
	cb.halfOpenCount.Store(0)
}

// ForceOpen forces the circuit into the open state.
func (cb *CircuitBreaker) ForceOpen() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.lastFailure.Store(time.Now().UnixMilli())
	cb.halfOpenCount.Store(0)
	cb.setState(StateOpen)
}

// ForceClose forces the circuit into the closed state.
func (cb *CircuitBreaker) ForceClose() {
	cb.Reset()
}

func (cb *CircuitBreaker) allowRequest() bool {
	state := cb.State()

	switch state {
	case StateClosed:
		return true

	case StateOpen:
		lastFailure := time.UnixMilli(cb.lastFailure.Load())
		if time.Since(lastFailure) > cb.config.Timeout {
			cb.mu.Lock()
			if cb.State() == StateOpen {
				cb.setState(StateHalfOpen)
				cb.successes.Store(0)
				cb.halfOpenCount.Store(0)
			}
			cb.mu.Unlock()
			return cb.reserveHalfOpen()
		}
		return false

	case StateHalfOpen:
		return cb.reserveHalfOpen()
	}

	return false
}

func (cb *CircuitBreaker) reserveHalfOpen() bool {
	if cb.config.MaxRequests <= 0 {
		return true
	}
	for {
		cur := cb.halfOpenCount.Load()
		if cur >= cb.config.MaxRequests {
			return false
		}
		if cb.halfOpenCount.CompareAndSwap(cur, cur+1) {
			return true
		}
	}
}

func (cb *CircuitBreaker) recordSuccess() {
	state := cb.State()

	switch state {
	case StateClosed:
		cb.failures.Store(0)

	case StateHalfOpen:
		successes := cb.successes.Add(1)
		if successes >= cb.config.SuccessThreshold {
			cb.mu.Lock()
			if cb.State() == StateHalfOpen {
				cb.setState(StateClosed)
				cb.failures.Store(0)
				cb.halfOpenCount.Store(0)
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
			}
			cb.mu.Unlock()
		}

	case StateHalfOpen:
		cb.mu.Lock()
		if cb.State() == StateHalfOpen {
			cb.setState(StateOpen)
			cb.halfOpenCount.Store(0)
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

var _ Breaker = (*CircuitBreaker)(nil)

package loadbalancer

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/resilience"
)

// Ensure ResilientManager implements LoadBalancerManager at compile time.
var _ LoadBalancerManager = (*ResilientManager)(nil)

// ResilientConfig configures the resilient load balancer manager wrapper.
type ResilientConfig struct {
	CircuitBreakerEnabled   bool
	CircuitBreakerThreshold int64
	CircuitBreakerTimeout   time.Duration

	RetryEnabled     bool
	RetryMaxAttempts int
	RetryBackoff     time.Duration
}

// ResilientManager wraps a LoadBalancerManager with circuit breaker and retry.
// NotFound / conflict / invalid-argument outcomes are not retried and do not open the circuit.
type ResilientManager struct {
	next     LoadBalancerManager
	cb       *resilience.CircuitBreaker
	retryCfg resilience.RetryConfig
}

// NewResilientManager wraps next with resilience features.
func NewResilientManager(next LoadBalancerManager, cfg ResilientConfig) *ResilientManager {
	rm := &ResilientManager{next: next}

	if cfg.CircuitBreakerEnabled {
		threshold := cfg.CircuitBreakerThreshold
		if threshold <= 0 {
			threshold = 5
		}
		timeout := cfg.CircuitBreakerTimeout
		if timeout <= 0 {
			timeout = 30 * time.Second
		}
		rm.cb = resilience.NewCircuitBreaker(resilience.CircuitBreakerConfig{
			Name:             "loadbalancer",
			FailureThreshold: threshold,
			SuccessThreshold: 2,
			Timeout:          timeout,
		})
	}

	if cfg.RetryEnabled && cfg.RetryMaxAttempts > 0 {
		backoff := cfg.RetryBackoff
		if backoff <= 0 {
			backoff = 100 * time.Millisecond
		}
		rm.retryCfg = resilience.RetryConfig{
			MaxAttempts:    cfg.RetryMaxAttempts,
			InitialBackoff: backoff,
			MaxBackoff:     5 * time.Second,
			Multiplier:     2.0,
			Jitter:         0.1,
			RetryIf: func(err error) bool {
				return err != nil && !isExpectedLBErr(err)
			},
		}
	}

	return rm
}

func isExpectedLBErr(err error) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, ErrLoadBalancerNotFound),
		errors.Is(err, ErrListenerNotFound),
		errors.Is(err, ErrTargetPoolNotFound),
		errors.Is(err, ErrTargetNotFound),
		errors.Is(err, ErrRuleNotFound),
		errors.Is(err, ErrTargetAlreadyRegistered),
		errors.Is(err, ErrInvalidProtocol),
		errors.Is(err, ErrInvalidPort),
		errors.Is(err, ErrLoadBalancerInUse),
		errors.Is(err, ErrTargetPoolInUse),
		errors.Is(err, ErrUnsupportedAlgorithm):
		return true
	}
	return false
}

func (m *ResilientManager) execute(ctx context.Context, fn resilience.Executor) error {
	operation := fn
	if m.cb != nil {
		cbFn := operation
		operation = func(ctx context.Context) error {
			var opErr error
			cbErr := m.cb.Execute(ctx, func(ctx context.Context) error {
				opErr = cbFn(ctx)
				if isExpectedLBErr(opErr) {
					return nil
				}
				return opErr
			})
			if cbErr != nil {
				return cbErr
			}
			return opErr
		}
	}
	if m.retryCfg.MaxAttempts > 0 {
		return resilience.Retry(ctx, m.retryCfg, operation)
	}
	return operation(ctx)
}

// CreateLoadBalancer runs CreateLoadBalancer with resilience.
func (m *ResilientManager) CreateLoadBalancer(ctx context.Context, opts CreateLoadBalancerOptions) (*LoadBalancer, error) {
	var lb *LoadBalancer
	err := m.execute(ctx, func(ctx context.Context) error {
		var e error
		lb, e = m.next.CreateLoadBalancer(ctx, opts)
		return e
	})
	return lb, err
}

// GetLoadBalancer runs GetLoadBalancer with resilience.
func (m *ResilientManager) GetLoadBalancer(ctx context.Context, id string) (*LoadBalancer, error) {
	var lb *LoadBalancer
	err := m.execute(ctx, func(ctx context.Context) error {
		var e error
		lb, e = m.next.GetLoadBalancer(ctx, id)
		return e
	})
	return lb, err
}

// ListLoadBalancers runs ListLoadBalancers with resilience.
func (m *ResilientManager) ListLoadBalancers(ctx context.Context) ([]*LoadBalancer, error) {
	var list []*LoadBalancer
	err := m.execute(ctx, func(ctx context.Context) error {
		var e error
		list, e = m.next.ListLoadBalancers(ctx)
		return e
	})
	return list, err
}

// DeleteLoadBalancer runs DeleteLoadBalancer with resilience.
func (m *ResilientManager) DeleteLoadBalancer(ctx context.Context, id string) error {
	return m.execute(ctx, func(ctx context.Context) error {
		return m.next.DeleteLoadBalancer(ctx, id)
	})
}

// CreateListener runs CreateListener with resilience.
func (m *ResilientManager) CreateListener(ctx context.Context, opts CreateListenerOptions) (*Listener, error) {
	var l *Listener
	err := m.execute(ctx, func(ctx context.Context) error {
		var e error
		l, e = m.next.CreateListener(ctx, opts)
		return e
	})
	return l, err
}

// DeleteListener runs DeleteListener with resilience.
func (m *ResilientManager) DeleteListener(ctx context.Context, loadBalancerID, listenerID string) error {
	return m.execute(ctx, func(ctx context.Context) error {
		return m.next.DeleteListener(ctx, loadBalancerID, listenerID)
	})
}

// CreateTargetPool runs CreateTargetPool with resilience.
func (m *ResilientManager) CreateTargetPool(ctx context.Context, opts CreateTargetPoolOptions) (*TargetPool, error) {
	var pool *TargetPool
	err := m.execute(ctx, func(ctx context.Context) error {
		var e error
		pool, e = m.next.CreateTargetPool(ctx, opts)
		return e
	})
	return pool, err
}

// GetTargetPool runs GetTargetPool with resilience.
func (m *ResilientManager) GetTargetPool(ctx context.Context, id string) (*TargetPool, error) {
	var pool *TargetPool
	err := m.execute(ctx, func(ctx context.Context) error {
		var e error
		pool, e = m.next.GetTargetPool(ctx, id)
		return e
	})
	return pool, err
}

// DeleteTargetPool runs DeleteTargetPool with resilience.
func (m *ResilientManager) DeleteTargetPool(ctx context.Context, id string) error {
	return m.execute(ctx, func(ctx context.Context) error {
		return m.next.DeleteTargetPool(ctx, id)
	})
}

// AddTarget runs AddTarget with resilience.
func (m *ResilientManager) AddTarget(ctx context.Context, poolID string, target Target) error {
	return m.execute(ctx, func(ctx context.Context) error {
		return m.next.AddTarget(ctx, poolID, target)
	})
}

// RemoveTarget runs RemoveTarget with resilience.
func (m *ResilientManager) RemoveTarget(ctx context.Context, poolID, targetID string) error {
	return m.execute(ctx, func(ctx context.Context) error {
		return m.next.RemoveTarget(ctx, poolID, targetID)
	})
}

// GetTargetHealth runs GetTargetHealth with resilience.
func (m *ResilientManager) GetTargetHealth(ctx context.Context, poolID string) ([]*Target, error) {
	var targets []*Target
	err := m.execute(ctx, func(ctx context.Context) error {
		var e error
		targets, e = m.next.GetTargetHealth(ctx, poolID)
		return e
	})
	return targets, err
}

// AddRule runs AddRule with resilience.
func (m *ResilientManager) AddRule(ctx context.Context, listenerID string, rule Rule) (*Rule, error) {
	var out *Rule
	err := m.execute(ctx, func(ctx context.Context) error {
		var e error
		out, e = m.next.AddRule(ctx, listenerID, rule)
		return e
	})
	return out, err
}

// RemoveRule runs RemoveRule with resilience.
func (m *ResilientManager) RemoveRule(ctx context.Context, listenerID, ruleID string) error {
	return m.execute(ctx, func(ctx context.Context) error {
		return m.next.RemoveRule(ctx, listenerID, ruleID)
	})
}

// Unwrap returns the underlying manager.
func (m *ResilientManager) Unwrap() LoadBalancerManager {
	return m.next
}

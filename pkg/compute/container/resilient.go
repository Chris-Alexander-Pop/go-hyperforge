package container

import (
	"context"
	"io"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/resilience"
)

// ResilientConfig configures the resilient container runtime wrapper.
type ResilientConfig struct {
	CircuitBreakerEnabled   bool
	CircuitBreakerThreshold int64
	CircuitBreakerTimeout   time.Duration

	RetryEnabled     bool
	RetryMaxAttempts int
	RetryBackoff     time.Duration
}

// ResilientRuntime wraps a ContainerRuntime with circuit breaker and retry.
// NotFound / conflict / invalid-argument outcomes are not retried and do not
// count as circuit failures.
type ResilientRuntime struct {
	next     ContainerRuntime
	cb       *resilience.CircuitBreaker
	retryCfg resilience.RetryConfig
}

var _ ContainerRuntime = (*ResilientRuntime)(nil)

// NewResilientRuntime wraps a runtime (e.g. k8s or fargate) with resilience.
func NewResilientRuntime(next ContainerRuntime, cfg ResilientConfig) *ResilientRuntime {
	rr := &ResilientRuntime{next: next}

	if cfg.CircuitBreakerEnabled {
		threshold := cfg.CircuitBreakerThreshold
		if threshold <= 0 {
			threshold = 5
		}
		timeout := cfg.CircuitBreakerTimeout
		if timeout <= 0 {
			timeout = 30 * time.Second
		}
		rr.cb = resilience.NewCircuitBreaker(resilience.CircuitBreakerConfig{
			Name:             "compute-container",
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
		rr.retryCfg = resilience.RetryConfig{
			MaxAttempts:    cfg.RetryMaxAttempts,
			InitialBackoff: backoff,
			MaxBackoff:     5 * time.Second,
			Multiplier:     2.0,
			RetryIf: func(err error) bool {
				if err == nil {
					return false
				}
				switch err {
				case ErrContainerNotFound, ErrImageNotFound, ErrContainerNotRunning,
					ErrContainerAlreadyRunning, ErrInvalidConfig, ErrNameConflict:
					return false
				}
				return true
			},
		}
	}

	return rr
}

func (r *ResilientRuntime) execute(ctx context.Context, fn resilience.Executor) error {
	operation := fn

	if r.cb != nil {
		cbFn := operation
		operation = func(ctx context.Context) error {
			var opErr error
			cbErr := r.cb.Execute(ctx, func(ctx context.Context) error {
				opErr = cbFn(ctx)
				if isExpectedContainerErr(opErr) {
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

	if r.retryCfg.MaxAttempts > 0 {
		return resilience.Retry(ctx, r.retryCfg, operation)
	}
	return operation(ctx)
}

func isExpectedContainerErr(err error) bool {
	switch err {
	case ErrContainerNotFound, ErrImageNotFound, ErrContainerNotRunning,
		ErrContainerAlreadyRunning, ErrInvalidConfig, ErrNameConflict:
		return true
	}
	return false
}

func (r *ResilientRuntime) Create(ctx context.Context, opts CreateOptions) (*Container, error) {
	var c *Container
	err := r.execute(ctx, func(ctx context.Context) error {
		var e error
		c, e = r.next.Create(ctx, opts)
		return e
	})
	return c, err
}

func (r *ResilientRuntime) Get(ctx context.Context, containerID string) (*Container, error) {
	var c *Container
	err := r.execute(ctx, func(ctx context.Context) error {
		var e error
		c, e = r.next.Get(ctx, containerID)
		return e
	})
	return c, err
}

func (r *ResilientRuntime) List(ctx context.Context, opts ListOptions) ([]*Container, error) {
	var list []*Container
	err := r.execute(ctx, func(ctx context.Context) error {
		var e error
		list, e = r.next.List(ctx, opts)
		return e
	})
	return list, err
}

func (r *ResilientRuntime) Start(ctx context.Context, containerID string) error {
	return r.execute(ctx, func(ctx context.Context) error {
		return r.next.Start(ctx, containerID)
	})
}

func (r *ResilientRuntime) Stop(ctx context.Context, containerID string, timeout time.Duration) error {
	return r.execute(ctx, func(ctx context.Context) error {
		return r.next.Stop(ctx, containerID, timeout)
	})
}

func (r *ResilientRuntime) Kill(ctx context.Context, containerID string, signal string) error {
	return r.execute(ctx, func(ctx context.Context) error {
		return r.next.Kill(ctx, containerID, signal)
	})
}

func (r *ResilientRuntime) Remove(ctx context.Context, containerID string, force bool) error {
	return r.execute(ctx, func(ctx context.Context) error {
		return r.next.Remove(ctx, containerID, force)
	})
}

func (r *ResilientRuntime) Logs(ctx context.Context, containerID string, follow bool) (io.ReadCloser, error) {
	var rc io.ReadCloser
	err := r.execute(ctx, func(ctx context.Context) error {
		var e error
		rc, e = r.next.Logs(ctx, containerID, follow)
		return e
	})
	return rc, err
}

func (r *ResilientRuntime) Exec(ctx context.Context, containerID string, opts ExecOptions) (*ExecResult, error) {
	var res *ExecResult
	err := r.execute(ctx, func(ctx context.Context) error {
		var e error
		res, e = r.next.Exec(ctx, containerID, opts)
		return e
	})
	return res, err
}

func (r *ResilientRuntime) Wait(ctx context.Context, containerID string) (int, error) {
	var code int
	err := r.execute(ctx, func(ctx context.Context) error {
		var e error
		code, e = r.next.Wait(ctx, containerID)
		return e
	})
	return code, err
}

func (r *ResilientRuntime) Stats(ctx context.Context, containerID string) (*ContainerStats, error) {
	var stats *ContainerStats
	err := r.execute(ctx, func(ctx context.Context) error {
		var e error
		stats, e = r.next.Stats(ctx, containerID)
		return e
	})
	return stats, err
}

// Unwrap returns the underlying runtime.
func (r *ResilientRuntime) Unwrap() ContainerRuntime {
	return r.next
}

// CircuitBreakerState returns the current circuit breaker state.
func (r *ResilientRuntime) CircuitBreakerState() resilience.State {
	if r.cb == nil {
		return ""
	}
	return r.cb.State()
}

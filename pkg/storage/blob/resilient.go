package blob

import (
	"context"
	"io"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/resilience"
)

// ResilientConfig configures the resilient blob store wrapper.
type ResilientConfig struct {
	CircuitBreakerEnabled   bool          `env:"BLOB_CB_ENABLED" env-default:"true"`
	CircuitBreakerThreshold int64         `env:"BLOB_CB_THRESHOLD" env-default:"5"`
	CircuitBreakerTimeout   time.Duration `env:"BLOB_CB_TIMEOUT" env-default:"30s"`

	RetryEnabled     bool          `env:"BLOB_RETRY_ENABLED" env-default:"true"`
	RetryMaxAttempts int           `env:"BLOB_RETRY_MAX" env-default:"3"`
	RetryBackoff     time.Duration `env:"BLOB_RETRY_BACKOFF" env-default:"100ms"`
}

// ResilientStore wraps a Store with circuit breaker and retry support.
// Missing blobs (NotFound) are expected outcomes: they are not retried and do
// not count toward the circuit breaker failure threshold.
type ResilientStore struct {
	store    Store
	cb       *resilience.CircuitBreaker
	retryCfg resilience.RetryConfig
}

// Ensure ResilientStore implements Store.
var _ Store = (*ResilientStore)(nil)

// NewResilientStore wraps a store with resilience features for cloud I/O.
func NewResilientStore(store Store, cfg ResilientConfig) *ResilientStore {
	rs := &ResilientStore{store: store}

	if cfg.CircuitBreakerEnabled {
		rs.cb = resilience.NewCircuitBreaker(resilience.CircuitBreakerConfig{
			Name:             "blob",
			FailureThreshold: cfg.CircuitBreakerThreshold,
			SuccessThreshold: 2,
			Timeout:          cfg.CircuitBreakerTimeout,
		})
	}

	if cfg.RetryEnabled {
		rs.retryCfg = resilience.RetryConfig{
			MaxAttempts:    cfg.RetryMaxAttempts,
			InitialBackoff: cfg.RetryBackoff,
			MaxBackoff:     5 * time.Second,
			Multiplier:     2.0,
			RetryIf: func(err error) bool {
				return err != nil && !IsNotFound(err) && !IsInvalidArgument(err)
			},
		}
	}

	return rs
}

func (rs *ResilientStore) Upload(ctx context.Context, key string, data io.Reader) error {
	return rs.execute(ctx, func(ctx context.Context) error {
		return rs.store.Upload(ctx, key, data)
	})
}

func (rs *ResilientStore) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	var rc io.ReadCloser
	err := rs.execute(ctx, func(ctx context.Context) error {
		var err error
		rc, err = rs.store.Download(ctx, key)
		return err
	})
	return rc, err
}

func (rs *ResilientStore) Delete(ctx context.Context, key string) error {
	return rs.execute(ctx, func(ctx context.Context) error {
		return rs.store.Delete(ctx, key)
	})
}

func (rs *ResilientStore) URL(key string) string {
	return rs.store.URL(key)
}

func (rs *ResilientStore) execute(ctx context.Context, fn resilience.Executor) error {
	operation := fn

	// NotFound is returned to the caller but recorded as success so misses
	// do not open the circuit.
	if rs.cb != nil {
		cbFn := operation
		operation = func(ctx context.Context) error {
			var opErr error
			cbErr := rs.cb.Execute(ctx, func(ctx context.Context) error {
				opErr = cbFn(ctx)
				if IsNotFound(opErr) {
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

	if rs.retryCfg.MaxAttempts > 0 {
		return resilience.Retry(ctx, rs.retryCfg, operation)
	}

	return operation(ctx)
}

// Unwrap returns the underlying store.
func (rs *ResilientStore) Unwrap() Store {
	return rs.store
}

// CircuitBreakerState returns the current circuit breaker state.
func (rs *ResilientStore) CircuitBreakerState() resilience.State {
	if rs.cb == nil {
		return ""
	}
	return rs.cb.State()
}

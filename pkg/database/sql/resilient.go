package sql

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/resilience"
	"gorm.io/gorm"
)

// ResilientConfig configures the resilient SQL wrapper.
type ResilientConfig struct {
	CircuitBreakerEnabled   bool          `env:"SQL_CB_ENABLED" env-default:"true"`
	CircuitBreakerThreshold int64         `env:"SQL_CB_THRESHOLD" env-default:"5"`
	CircuitBreakerTimeout   time.Duration `env:"SQL_CB_TIMEOUT" env-default:"30s"`

	RetryEnabled     bool          `env:"SQL_RETRY_ENABLED" env-default:"true"`
	RetryMaxAttempts int           `env:"SQL_RETRY_MAX" env-default:"3"`
	RetryBackoff     time.Duration `env:"SQL_RETRY_BACKOFF" env-default:"100ms"`
}

// ResilientSQL wraps an SQL backend with circuit breaker and retry support.
// Get remains a pass-through (it does not return an error); use Execute as the
// primary entry point for fallible work that should be retried / circuit-broken.
type ResilientSQL struct {
	next     SQL
	cb       *resilience.CircuitBreaker
	retryCfg resilience.RetryConfig
}

// Ensure ResilientSQL implements SQL.
var _ SQL = (*ResilientSQL)(nil)

// NewResilientSQL wraps a SQL backend with resilience features.
func NewResilientSQL(next SQL, cfg ResilientConfig) *ResilientSQL {
	rs := &ResilientSQL{next: next}

	if cfg.CircuitBreakerEnabled {
		rs.cb = resilience.NewCircuitBreaker(resilience.CircuitBreakerConfig{
			Name:             "sql",
			FailureThreshold: cfg.CircuitBreakerThreshold,
			SuccessThreshold: 2,
			Timeout:          cfg.CircuitBreakerTimeout,
		})
	}

	if cfg.RetryEnabled && cfg.RetryMaxAttempts > 0 {
		rs.retryCfg = resilience.RetryConfig{
			MaxAttempts:    cfg.RetryMaxAttempts,
			InitialBackoff: cfg.RetryBackoff,
			MaxBackoff:     10 * time.Second,
			Multiplier:     2.0,
			Jitter:         0.1,
			RetryIf:        func(err error) bool { return err != nil },
		}
	}

	return rs
}

// Get returns the primary database connection (pass-through; no error surface).
func (r *ResilientSQL) Get(ctx context.Context) *gorm.DB {
	return r.next.Get(ctx)
}

// GetShard resolves a shard connection with retry/circuit-breaker protection.
func (r *ResilientSQL) GetShard(ctx context.Context, key string) (*gorm.DB, error) {
	var db *gorm.DB
	err := r.execute(ctx, func(ctx context.Context) error {
		var err error
		db, err = r.next.GetShard(ctx, key)
		return err
	})
	return db, err
}

// Close releases resources with retry/circuit-breaker protection.
func (r *ResilientSQL) Close() error {
	return r.execute(context.Background(), func(ctx context.Context) error {
		return r.next.Close()
	})
}

// Execute runs fn against the primary connection under retry and optional
// circuit breaking. This is the preferred entry point for fallible SQL work.
func (r *ResilientSQL) Execute(ctx context.Context, fn func(ctx context.Context, db *gorm.DB) error) error {
	return r.execute(ctx, func(ctx context.Context) error {
		return fn(ctx, r.next.Get(ctx))
	})
}

// Unwrap returns the underlying SQL backend.
func (r *ResilientSQL) Unwrap() SQL {
	return r.next
}

// CircuitBreakerState returns the current circuit breaker state.
func (r *ResilientSQL) CircuitBreakerState() resilience.State {
	if r.cb == nil {
		return ""
	}
	return r.cb.State()
}

func (r *ResilientSQL) execute(ctx context.Context, fn resilience.Executor) error {
	operation := fn

	if r.cb != nil {
		cbFn := operation
		operation = func(ctx context.Context) error {
			return r.cb.Execute(ctx, cbFn)
		}
	}

	if r.retryCfg.MaxAttempts > 0 {
		return resilience.Retry(ctx, r.retryCfg, operation)
	}

	return operation(ctx)
}

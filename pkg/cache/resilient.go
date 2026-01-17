package cache

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/resilience"
)

// ResilientCache wraps a Cache with circuit breaker and retry support.
// This prevents cache failures from cascading and provides automatic recovery.
type ResilientCache struct {
	cache    Cache
	cb       *resilience.CircuitBreaker
	retryCfg resilience.RetryConfig
}

// ResilientConfig configures the resilient cache wrapper.
type ResilientConfig struct {
	// Circuit breaker settings
	CircuitBreakerEnabled   bool          `env:"CACHE_CB_ENABLED" env-default:"true"`
	CircuitBreakerThreshold int64         `env:"CACHE_CB_THRESHOLD" env-default:"5"`
	CircuitBreakerTimeout   time.Duration `env:"CACHE_CB_TIMEOUT" env-default:"30s"`

	// Retry settings
	RetryEnabled     bool          `env:"CACHE_RETRY_ENABLED" env-default:"true"`
	RetryMaxAttempts int           `env:"CACHE_RETRY_MAX" env-default:"2"`
	RetryBackoff     time.Duration `env:"CACHE_RETRY_BACKOFF" env-default:"50ms"`
}

// NewResilientCache wraps a cache with resilience features.
func NewResilientCache(cache Cache, cfg ResilientConfig) *ResilientCache {
	rc := &ResilientCache{
		cache: cache,
	}

	if cfg.CircuitBreakerEnabled {
		rc.cb = resilience.NewCircuitBreaker(resilience.CircuitBreakerConfig{
			Name:             "cache",
			FailureThreshold: cfg.CircuitBreakerThreshold,
			SuccessThreshold: 2,
			Timeout:          cfg.CircuitBreakerTimeout,
		})
	}

	if cfg.RetryEnabled {
		rc.retryCfg = resilience.RetryConfig{
			MaxAttempts:    cfg.RetryMaxAttempts,
			InitialBackoff: cfg.RetryBackoff,
			MaxBackoff:     time.Second,
			Multiplier:     2.0,
		}
	}

	return rc
}

func (rc *ResilientCache) Get(ctx context.Context, key string, dest interface{}) error {
	return rc.execute(ctx, func(ctx context.Context) error {
		return rc.cache.Get(ctx, key, dest)
	})
}

func (rc *ResilientCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return rc.execute(ctx, func(ctx context.Context) error {
		return rc.cache.Set(ctx, key, value, ttl)
	})
}

func (rc *ResilientCache) Delete(ctx context.Context, key string) error {
	return rc.execute(ctx, func(ctx context.Context) error {
		return rc.cache.Delete(ctx, key)
	})
}

func (rc *ResilientCache) Incr(ctx context.Context, key string, delta int64) (int64, error) {
	var result int64
	err := rc.execute(ctx, func(ctx context.Context) error {
		var err error
		result, err = rc.cache.Incr(ctx, key, delta)
		return err
	})
	return result, err
}

func (rc *ResilientCache) Close() error {
	return rc.cache.Close()
}

func (rc *ResilientCache) execute(ctx context.Context, fn resilience.Executor) error {
	operation := fn

	// Wrap with circuit breaker if enabled
	if rc.cb != nil {
		cbFn := operation
		operation = func(ctx context.Context) error {
			return rc.cb.Execute(ctx, cbFn)
		}
	}

	// Wrap with retry if enabled
	if rc.retryCfg.MaxAttempts > 0 {
		return resilience.Retry(ctx, rc.retryCfg, operation)
	}

	return operation(ctx)
}

// Unwrap returns the underlying cache.
func (rc *ResilientCache) Unwrap() Cache {
	return rc.cache
}

// CircuitBreakerState returns the current circuit breaker state.
func (rc *ResilientCache) CircuitBreakerState() resilience.State {
	if rc.cb == nil {
		return ""
	}
	return rc.cb.State()
}

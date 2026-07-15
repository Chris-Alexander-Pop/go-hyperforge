package resilience

import "time"

// Config is an env-tagged resilience configuration covering circuit breaker,
// retry, and hedge defaults. Use CircuitBreaker() / Retry() helpers to derive
// the concrete configs consumed by NewCircuitBreaker and Retry.
type Config struct {
	// Name identifies the circuit breaker (logging/metrics).
	Name string `env:"RESILIENCE_NAME" env-default:"default"`

	// FailureThreshold opens the circuit after this many consecutive failures.
	FailureThreshold int64 `env:"RESILIENCE_FAILURE_THRESHOLD" env-default:"5"`

	// SuccessThreshold closes the circuit after this many half-open successes.
	SuccessThreshold int64 `env:"RESILIENCE_SUCCESS_THRESHOLD" env-default:"2"`

	// Timeout is how long the circuit stays open before half-open probes.
	Timeout time.Duration `env:"RESILIENCE_TIMEOUT" env-default:"30s"`

	// MaxRequests caps concurrent half-open probes (0 = unlimited).
	MaxRequests int64 `env:"RESILIENCE_MAX_REQUESTS" env-default:"0"`

	// MaxAttempts is the maximum retry attempts including the first.
	MaxAttempts int `env:"RESILIENCE_MAX_ATTEMPTS" env-default:"3"`

	// InitialBackoff is the first retry backoff.
	InitialBackoff time.Duration `env:"RESILIENCE_INITIAL_BACKOFF" env-default:"100ms"`

	// MaxBackoff caps exponential backoff.
	MaxBackoff time.Duration `env:"RESILIENCE_MAX_BACKOFF" env-default:"10s"`

	// Multiplier grows backoff between retries.
	Multiplier float64 `env:"RESILIENCE_MULTIPLIER" env-default:"2"`

	// Jitter adds randomness to backoff (0–1 fraction).
	Jitter float64 `env:"RESILIENCE_JITTER" env-default:"0.1"`

	// HedgeDelay is the delay before starting a speculative hedge request.
	HedgeDelay time.Duration `env:"RESILIENCE_HEDGE_DELAY" env-default:"50ms"`

	// BulkheadMaxConcurrent caps concurrent work in a bulkhead (0 = unset).
	BulkheadMaxConcurrent int64 `env:"RESILIENCE_BULKHEAD_MAX" env-default:"0"`
}

// DefaultConfig returns env-default values without reading the environment.
func DefaultConfig() Config {
	return Config{
		Name:                  "default",
		FailureThreshold:      5,
		SuccessThreshold:      2,
		Timeout:               30 * time.Second,
		MaxRequests:           0,
		MaxAttempts:           3,
		InitialBackoff:        100 * time.Millisecond,
		MaxBackoff:            10 * time.Second,
		Multiplier:            2.0,
		Jitter:                0.1,
		HedgeDelay:            50 * time.Millisecond,
		BulkheadMaxConcurrent: 0,
	}
}

// CircuitBreaker returns a CircuitBreakerConfig derived from Config.
func (c Config) CircuitBreaker() CircuitBreakerConfig {
	name := c.Name
	if name == "" {
		name = "default"
	}
	return CircuitBreakerConfig{
		Name:             name,
		FailureThreshold: c.FailureThreshold,
		SuccessThreshold: c.SuccessThreshold,
		Timeout:          c.Timeout,
		MaxRequests:      c.MaxRequests,
	}
}

// Retry returns a RetryConfig derived from Config.
func (c Config) Retry() RetryConfig {
	return RetryConfig{
		MaxAttempts:    c.MaxAttempts,
		InitialBackoff: c.InitialBackoff,
		MaxBackoff:     c.MaxBackoff,
		Multiplier:     c.Multiplier,
		Jitter:         c.Jitter,
		RetryIf:        func(err error) bool { return err != nil },
	}
}

// Bulkhead returns a BulkheadConfig derived from Config when MaxConcurrent > 0.
func (c Config) Bulkhead() BulkheadConfig {
	return BulkheadConfig{
		Name:          c.Name,
		MaxConcurrent: c.BulkheadMaxConcurrent,
	}
}

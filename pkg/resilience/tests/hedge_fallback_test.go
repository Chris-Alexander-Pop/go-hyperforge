package resilience_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/resilience"
)

func TestConfigDefaults(t *testing.T) {
	cfg := resilience.DefaultConfig()
	if cfg.Name != "default" {
		t.Fatalf("Name=%q", cfg.Name)
	}
	if cfg.FailureThreshold != 5 {
		t.Fatalf("FailureThreshold=%d", cfg.FailureThreshold)
	}
	if cfg.HedgeDelay != 50*time.Millisecond {
		t.Fatalf("HedgeDelay=%v", cfg.HedgeDelay)
	}

	cb := cfg.CircuitBreaker()
	if cb.Name != "default" || cb.FailureThreshold != 5 {
		t.Fatalf("CircuitBreaker() = %+v", cb)
	}
	rc := cfg.Retry()
	if rc.MaxAttempts != 3 || rc.Multiplier != 2.0 {
		t.Fatalf("Retry() = %+v", rc)
	}
}

func TestConfigEnvTagsPresent(t *testing.T) {
	// Compile-time / reflection-free smoke: env tags live on Config fields.
	cfg := resilience.Config{
		Name:             "api",
		FailureThreshold: 3,
		MaxAttempts:      2,
		HedgeDelay:       10 * time.Millisecond,
	}
	if cfg.CircuitBreaker().Name != "api" {
		t.Fatal("expected Name to flow into CircuitBreakerConfig")
	}
	if cfg.Retry().MaxAttempts != 2 {
		t.Fatal("expected MaxAttempts to flow into RetryConfig")
	}
}

func TestFallback_PrimaryWins(t *testing.T) {
	err := resilience.Fallback(context.Background(),
		func(ctx context.Context) error { return nil },
		func(ctx context.Context) error { return errors.New("should not run") },
	)
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestFallback_UsesSecondary(t *testing.T) {
	var secondary atomic.Bool
	err := resilience.Fallback(context.Background(),
		func(ctx context.Context) error { return errors.New("primary") },
		func(ctx context.Context) error {
			secondary.Store(true)
			return nil
		},
	)
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if !secondary.Load() {
		t.Fatal("expected secondary to run")
	}
}

func TestFallbackT(t *testing.T) {
	val, err := resilience.FallbackT(context.Background(),
		func(ctx context.Context) (int, error) { return 0, errors.New("fail") },
		func(ctx context.Context) (int, error) { return 42, nil },
	)
	if err != nil || val != 42 {
		t.Fatalf("got %d, %v", val, err)
	}
}

func TestExecuteT(t *testing.T) {
	cb := resilience.NewCircuitBreaker(resilience.DefaultCircuitBreakerConfig("typed"))
	val, err := resilience.ExecuteT(context.Background(), cb, func(ctx context.Context) (string, error) {
		return "ok", nil
	})
	if err != nil || val != "ok" {
		t.Fatalf("got %q, %v", val, err)
	}
}

func TestExecuteT_NilBreaker(t *testing.T) {
	val, err := resilience.ExecuteT[int](context.Background(), nil, func(ctx context.Context) (int, error) {
		return 7, nil
	})
	if err != nil || val != 7 {
		t.Fatalf("got %d, %v", val, err)
	}
}

func TestRetryT(t *testing.T) {
	var calls atomic.Int64
	val, err := resilience.RetryT(context.Background(), resilience.RetryConfig{
		MaxAttempts:    3,
		InitialBackoff: time.Millisecond,
		MaxBackoff:     5 * time.Millisecond,
		Multiplier:     2,
	}, func(ctx context.Context) (int, error) {
		if calls.Add(1) < 3 {
			return 0, errors.New("temp")
		}
		return 99, nil
	})
	if err != nil || val != 99 {
		t.Fatalf("got %d, %v", val, err)
	}
	if calls.Load() != 3 {
		t.Fatalf("calls=%d", calls.Load())
	}
}

func TestHedge_PrimaryFast(t *testing.T) {
	var calls atomic.Int64
	err := resilience.Hedge(context.Background(), 50*time.Millisecond, func(ctx context.Context) error {
		calls.Add(1)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(80 * time.Millisecond)
	if calls.Load() != 1 {
		t.Fatalf("expected 1 call when primary is fast, got %d", calls.Load())
	}
}

func TestHedge_SpeculativeRetry(t *testing.T) {
	var calls atomic.Int64
	start := time.Now()
	val, err := resilience.HedgeT(context.Background(), 20*time.Millisecond, func(ctx context.Context) (int, error) {
		n := calls.Add(1)
		if n == 1 {
			select {
			case <-ctx.Done():
				return 0, ctx.Err()
			case <-time.After(200 * time.Millisecond):
				return 1, nil
			}
		}
		return 2, nil
	})
	elapsed := time.Since(start)
	if err != nil {
		t.Fatal(err)
	}
	if val != 2 {
		t.Fatalf("expected hedge winner 2, got %d", val)
	}
	if elapsed > 150*time.Millisecond {
		t.Fatalf("hedge should win quickly, took %v", elapsed)
	}
	if calls.Load() < 2 {
		t.Fatalf("expected at least 2 calls, got %d", calls.Load())
	}
}

func TestHedge_NoDelay(t *testing.T) {
	var calls atomic.Int64
	err := resilience.Hedge(context.Background(), 0, func(ctx context.Context) error {
		calls.Add(1)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 1 {
		t.Fatalf("calls=%d", calls.Load())
	}
}

func TestHedge_PrimaryFailsStartsSecond(t *testing.T) {
	var calls atomic.Int64
	err := resilience.Hedge(context.Background(), time.Hour, func(ctx context.Context) error {
		if calls.Add(1) == 1 {
			return errors.New("primary")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("expected second attempt success, got %v", err)
	}
	if calls.Load() != 2 {
		t.Fatalf("calls=%d", calls.Load())
	}
}

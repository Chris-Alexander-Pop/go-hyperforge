package resilience_test

import (
	"context"
	"testing"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/resilience"
)

func BenchmarkCircuitBreaker_Execute_Closed(b *testing.B) {
	cb := resilience.NewCircuitBreaker(resilience.CircuitBreakerConfig{
		Name:             "benchmark",
		FailureThreshold: 5,
		Timeout:          5 * time.Second,
	})

	ctx := context.Background()
	operation := func(ctx context.Context) error {
		return nil // Always succeeds
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cb.Execute(ctx, operation)
	}
}

func BenchmarkRetry_NoRetries(b *testing.B) {
	ctx := context.Background()
	cfg := resilience.RetryConfig{
		MaxAttempts:    3,
		InitialBackoff: time.Millisecond,
	}

	operation := func(ctx context.Context) error {
		return nil // Always succeeds
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = resilience.Retry(ctx, cfg, operation)
	}
}

func BenchmarkCircuitBreaker_Parallel(b *testing.B) {
	cb := resilience.NewCircuitBreaker(resilience.CircuitBreakerConfig{
		Name:             "parallel-bench",
		FailureThreshold: 100,
		Timeout:          10 * time.Second,
	})

	ctx := context.Background()
	operation := func(ctx context.Context) error {
		return nil
	}

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = cb.Execute(ctx, operation)
		}
	})
}

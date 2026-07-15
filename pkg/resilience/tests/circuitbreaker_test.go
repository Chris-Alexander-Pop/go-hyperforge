package resilience_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/resilience"
)

func TestCircuitBreaker_HalfOpenFailure(t *testing.T) {
	cfg := resilience.CircuitBreakerConfig{
		Name:             "test-cb-fail",
		FailureThreshold: 1,
		SuccessThreshold: 2,
		Timeout:          50 * time.Millisecond,
	}
	cb := resilience.NewCircuitBreaker(cfg)
	ctx := context.Background()
	fail := errors.New("fail")

	// Trip to Open
	if err := cb.Execute(ctx, func(ctx context.Context) error { return fail }); err == nil {
		t.Error("Expected error from Execute")
	}
	if cb.State() != resilience.StateOpen {
		t.Fatalf("Failed to open circuit")
	}

	// Wait for Timeout (Ready for Half-Open)
	time.Sleep(100 * time.Millisecond)

	// Execute failure in Half-Open -> Trip back to Open immediately
	if err := cb.Execute(ctx, func(ctx context.Context) error { return fail }); err == nil {
		t.Error("Expected error from Execute")
	}

	if cb.State() != resilience.StateOpen {
		t.Errorf("Expected state Open after half-open failure, got %v", cb.State())
	}
}

func TestCircuitBreaker_Reset(t *testing.T) {
	cb := resilience.NewCircuitBreaker(resilience.DefaultCircuitBreakerConfig("reset-test"))
	// Instead of unexported setState, we simulate failure to open
	ctx := context.Background()
	fail := errors.New("fail")
	for i := 0; i < 5; i++ {
		_ = cb.Execute(ctx, func(ctx context.Context) error { return fail })
	}

	if cb.State() != resilience.StateOpen {
		t.Error("Expected Open state before Reset")
	}

	cb.Reset()
	if cb.State() != resilience.StateClosed {
		t.Error("Reset failed to close circuit")
	}
}

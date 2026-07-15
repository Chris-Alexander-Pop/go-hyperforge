package resilience_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/resilience"
)

func TestCircuitBreaker_StateTransitions(t *testing.T) {
	cfg := resilience.CircuitBreakerConfig{
		Name:             "test-cb",
		FailureThreshold: 2,
		SuccessThreshold: 1,
		Timeout:          100 * time.Millisecond,
	}
	cb := resilience.NewCircuitBreaker(cfg)

	ctx := context.Background()
	failErr := errors.New("failure")

	// 1. Initial State: Closed
	if cb.State() != resilience.StateClosed {
		t.Errorf("Expected state Closed, got %v", cb.State())
	}

	// 2. Failure 1: Still Closed
	if err := cb.Execute(ctx, func(ctx context.Context) error { return failErr }); err == nil {
		t.Error("Expected error from Execute")
	}
	if cb.State() != resilience.StateClosed {
		t.Errorf("Expected state Closed, got %v", cb.State())
	}

	// 3. Failure 2: Trip to Open
	if err := cb.Execute(ctx, func(ctx context.Context) error { return failErr }); err == nil {
		t.Error("Expected error from Execute")
	}
	if cb.State() != resilience.StateOpen {
		t.Errorf("Expected state Open, got %v", cb.State())
	}

	// 4. Request while Open: Fast Fail
	err := cb.Execute(ctx, func(ctx context.Context) error { return nil })
	if !errors.Is(err, resilience.ErrCircuitOpen) {
		t.Errorf("Expected ErrCircuitOpen, got %v", err)
	}

	// 5. Wait for Timeout -> Half-Open
	time.Sleep(150 * time.Millisecond)

	// The state transition happens lazily on the next Execute or allowRequest
	// We'll mimic a successful call
	err = cb.Execute(ctx, func(ctx context.Context) error { return nil })
	if err != nil {
		t.Errorf("Expected success in Half-Open, got %v", err)
	}

	// 6. Success -> Closed
	if cb.State() != resilience.StateClosed {
		t.Errorf("Expected state Closed after success, got %v", cb.State())
	}
}

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

package resilience_test

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/resilience"
)

func ExampleCircuitBreaker() {
	// Create a circuit breaker
	cb := resilience.NewCircuitBreaker(resilience.CircuitBreakerConfig{
		Name:             "my-service",
		FailureThreshold: 3,
		SuccessThreshold: 2,
		Timeout:          5 * time.Second,
	})

	ctx := context.Background()

	// Execute with circuit breaker protection
	err := cb.Execute(ctx, func(ctx context.Context) error {
		// Your operation here
		return nil
	})

	if err != nil {
		fmt.Println("Operation failed:", err)
	} else {
		fmt.Println("Operation succeeded")
	}
	// Output: Operation succeeded
}

func ExampleRetry() {
	ctx := context.Background()
	attempts := 0

	// Retry with exponential backoff
	err := resilience.Retry(ctx, resilience.RetryConfig{
		MaxAttempts:    3,
		InitialBackoff: 10 * time.Millisecond,
		MaxBackoff:     100 * time.Millisecond,
		Multiplier:     2.0,
	}, func(ctx context.Context) error {
		attempts++
		if attempts < 3 {
			return errors.New("temporary failure")
		}
		return nil
	})

	if err != nil {
		fmt.Println("All retries failed:", err)
	} else {
		fmt.Printf("Succeeded after %d attempts\n", attempts)
	}
	// Output: Succeeded after 3 attempts
}

func Example_circuitBreakerWithRetry() {
	cb := resilience.NewCircuitBreaker(resilience.CircuitBreakerConfig{
		Name:             "api-call",
		FailureThreshold: 5,
		Timeout:          10 * time.Second,
	})

	ctx := context.Background()

	// Combine circuit breaker with retry
	err := cb.Execute(ctx, func(ctx context.Context) error {
		return resilience.Retry(ctx, resilience.RetryConfig{
			MaxAttempts:    3,
			InitialBackoff: 50 * time.Millisecond,
		}, func(ctx context.Context) error {
			// Your API call here
			return nil
		})
	})

	fmt.Println(err == nil)
	// Output: true
}

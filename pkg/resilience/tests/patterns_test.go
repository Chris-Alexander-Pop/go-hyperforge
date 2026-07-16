package resilience_test

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	apperrors "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/resilience"
)

func TestBulkhead_LimitsConcurrency(t *testing.T) {
	bh := resilience.NewBulkhead(resilience.BulkheadConfig{
		Name:          "test",
		MaxConcurrent: 2,
	})

	var inFlight atomic.Int64
	var maxSeen atomic.Int64
	var wg sync.WaitGroup

	ctx := context.Background()
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = bh.Execute(ctx, func(ctx context.Context) error {
				cur := inFlight.Add(1)
				for {
					old := maxSeen.Load()
					if cur <= old || maxSeen.CompareAndSwap(old, cur) {
						break
					}
				}
				time.Sleep(20 * time.Millisecond)
				inFlight.Add(-1)
				return nil
			})
		}()
	}
	wg.Wait()

	if maxSeen.Load() > 2 {
		t.Fatalf("expected max concurrency <= 2, got %d", maxSeen.Load())
	}
}

func TestBulkhead_TryExecuteFull(t *testing.T) {
	bh := resilience.NewBulkhead(resilience.BulkheadConfig{
		Name:          "try",
		MaxConcurrent: 1,
	})

	started := make(chan struct{})
	release := make(chan struct{})
	done := make(chan struct{})

	go func() {
		_ = bh.Execute(context.Background(), func(ctx context.Context) error {
			close(started)
			<-release
			return nil
		})
		close(done)
	}()

	<-started
	err := bh.TryExecute(context.Background(), func(ctx context.Context) error {
		return nil
	})
	if !errors.Is(err, resilience.ErrBulkheadFull) {
		t.Fatalf("expected ErrBulkheadFull, got %v", err)
	}
	close(release)
	<-done
}

func TestWithTimeout(t *testing.T) {
	fn := resilience.WithTimeout(30*time.Millisecond, func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(200 * time.Millisecond):
			return nil
		}
	})

	err := fn(context.Background())
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded, got %v", err)
	}
	if !apperrors.IsCode(err, apperrors.CodeDeadlineExceeded) {
		t.Fatalf("expected CodeDeadlineExceeded, got %q", apperrors.Code(err))
	}
}

func TestWithTimeout_Success(t *testing.T) {
	fn := resilience.WithTimeout(time.Second, func(ctx context.Context) error {
		return nil
	})
	if err := fn(context.Background()); err != nil {
		t.Fatalf("expected success, got %v", err)
	}
}

func TestWithTimeout_IgnoresContext(t *testing.T) {
	// fn blocks without watching ctx; WithTimeout must still return on deadline.
	fn := resilience.WithTimeout(40*time.Millisecond, func(ctx context.Context) error {
		time.Sleep(500 * time.Millisecond)
		return nil
	})
	start := time.Now()
	err := fn(context.Background())
	elapsed := time.Since(start)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded, got %v", err)
	}
	if !apperrors.IsCode(err, apperrors.CodeDeadlineExceeded) {
		t.Fatalf("expected CodeDeadlineExceeded, got %q", apperrors.Code(err))
	}
	if elapsed > 300*time.Millisecond {
		t.Fatalf("timeout not enforced promptly, took %v", elapsed)
	}
}

func TestWithTimeout_CodeStableUnderLoad(t *testing.T) {
	// Regression: when fn returns bare ctx.Err() on deadline, the done-channel
	// path used to skip CodeDeadlineExceeded wrapping (flake under load).
	const n = 200
	var wg sync.WaitGroup
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			fn := resilience.WithTimeout(2*time.Millisecond, func(ctx context.Context) error {
				<-ctx.Done()
				return ctx.Err()
			})
			errs <- fn(context.Background())
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("expected deadline exceeded, got %v", err)
		}
		if !apperrors.IsCode(err, apperrors.CodeDeadlineExceeded) {
			t.Fatalf("expected CodeDeadlineExceeded, got %q", apperrors.Code(err))
		}
	}
}

func TestWithTimeout_NonPositive(t *testing.T) {
	called := false
	fn := resilience.WithTimeout(0, func(ctx context.Context) error {
		called = true
		return nil
	})
	if err := fn(context.Background()); err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if !called {
		t.Fatal("expected fn to run when timeout <= 0")
	}
}

func TestRetryWithCircuitBreaker(t *testing.T) {
	cb := resilience.NewCircuitBreaker(resilience.CircuitBreakerConfig{
		Name:             "retry-cb",
		FailureThreshold: 5,
		Timeout:          time.Second,
	})

	cfg := resilience.RetryConfig{
		MaxAttempts:    3,
		InitialBackoff: time.Millisecond,
		MaxBackoff:     10 * time.Millisecond,
		Multiplier:     2,
	}

	var calls atomic.Int64
	err := resilience.RetryWithCircuitBreaker(context.Background(), cb, cfg, func(ctx context.Context) error {
		n := calls.Add(1)
		if n < 3 {
			return errors.New("temp")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if calls.Load() != 3 {
		t.Fatalf("expected 3 calls, got %d", calls.Load())
	}
}

func TestRetryWithCircuitBreaker_Open(t *testing.T) {
	cb := resilience.NewCircuitBreaker(resilience.CircuitBreakerConfig{
		Name:             "open-cb",
		FailureThreshold: 1,
		Timeout:          time.Hour,
	})

	cfg := resilience.RetryConfig{
		MaxAttempts:    3,
		InitialBackoff: time.Millisecond,
		RetryIf:        func(err error) bool { return !errors.Is(err, resilience.ErrCircuitOpen) },
	}

	_ = cb.Execute(context.Background(), func(ctx context.Context) error {
		return errors.New("fail")
	})

	err := resilience.RetryWithCircuitBreaker(context.Background(), cb, cfg, func(ctx context.Context) error {
		return nil
	})
	if !errors.Is(err, resilience.ErrCircuitOpen) {
		t.Fatalf("expected ErrCircuitOpen, got %v", err)
	}
}

func TestCircuitBreaker_ForceOpenClose(t *testing.T) {
	cb := resilience.NewCircuitBreaker(resilience.DefaultCircuitBreakerConfig("force"))
	cb.ForceOpen()
	if cb.State() != resilience.StateOpen {
		t.Fatalf("expected open, got %v", cb.State())
	}
	err := cb.Execute(context.Background(), func(ctx context.Context) error { return nil })
	if !errors.Is(err, resilience.ErrCircuitOpen) {
		t.Fatalf("expected ErrCircuitOpen, got %v", err)
	}
	cb.ForceClose()
	if cb.State() != resilience.StateClosed {
		t.Fatalf("expected closed, got %v", cb.State())
	}
}

func TestExponentialBackoff(t *testing.T) {
	d := resilience.ExponentialBackoff(0, 100*time.Millisecond, time.Second, 0)
	if d != 100*time.Millisecond {
		t.Fatalf("expected 100ms, got %v", d)
	}
	d = resilience.ExponentialBackoff(10, 100*time.Millisecond, 200*time.Millisecond, 0)
	if d != 200*time.Millisecond {
		t.Fatalf("expected capped 200ms, got %v", d)
	}
}

func TestInstrumentedCircuitBreaker(t *testing.T) {
	icb := resilience.NewInstrumentedBreakerFromConfig(resilience.CircuitBreakerConfig{
		Name:             "instr",
		FailureThreshold: 1,
		Timeout:          time.Hour,
	})
	_ = icb.Execute(context.Background(), func(ctx context.Context) error {
		return errors.New("fail")
	})
	if icb.State() != resilience.StateOpen {
		t.Fatalf("expected open, got %v", icb.State())
	}
	err := icb.Execute(context.Background(), func(ctx context.Context) error { return nil })
	if !errors.Is(err, resilience.ErrCircuitOpen) {
		t.Fatalf("expected ErrCircuitOpen, got %v", err)
	}
}

func TestCircuitBreaker_MaxRequests(t *testing.T) {
	cb := resilience.NewCircuitBreaker(resilience.CircuitBreakerConfig{
		Name:             "max-req",
		FailureThreshold: 1,
		SuccessThreshold: 5,
		Timeout:          20 * time.Millisecond,
		MaxRequests:      1,
	})
	ctx := context.Background()

	_ = cb.Execute(ctx, func(ctx context.Context) error { return errors.New("fail") })
	if cb.State() != resilience.StateOpen {
		t.Fatalf("expected open, got %v", cb.State())
	}

	time.Sleep(40 * time.Millisecond)

	started := make(chan struct{})
	release := make(chan struct{})
	done := make(chan error, 1)

	go func() {
		done <- cb.Execute(ctx, func(ctx context.Context) error {
			close(started)
			<-release
			return nil
		})
	}()

	<-started
	err := cb.Execute(ctx, func(ctx context.Context) error { return nil })
	if !errors.Is(err, resilience.ErrTooManyRequests) {
		t.Fatalf("expected ErrTooManyRequests, got %v", err)
	}
	if apperrors.HTTPStatus(err) != http.StatusTooManyRequests {
		t.Fatalf("expected HTTP 429, got %d", apperrors.HTTPStatus(err))
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatalf("probe failed: %v", err)
	}
}

func TestErrorCodes(t *testing.T) {
	if apperrors.HTTPStatus(resilience.ErrCircuitOpen) != http.StatusServiceUnavailable {
		t.Fatalf("circuit open should map to 503, got %d", apperrors.HTTPStatus(resilience.ErrCircuitOpen))
	}
	if !apperrors.IsCode(resilience.ErrCircuitOpen, apperrors.CodeUnavailable) {
		t.Fatalf("expected UNAVAILABLE code, got %q", apperrors.Code(resilience.ErrCircuitOpen))
	}
	if apperrors.HTTPStatus(resilience.ErrBulkheadFull) != http.StatusTooManyRequests {
		t.Fatalf("bulkhead full should map to 429, got %d", apperrors.HTTPStatus(resilience.ErrBulkheadFull))
	}
	if !apperrors.IsCode(resilience.ErrBulkheadFull, apperrors.CodeResourceExhausted) {
		t.Fatalf("expected RESOURCE_EXHAUSTED, got %q", apperrors.Code(resilience.ErrBulkheadFull))
	}
}

func TestRetrier(t *testing.T) {
	var calls atomic.Int64
	r := resilience.NewRetrier(resilience.RetryConfig{
		MaxAttempts:    3,
		InitialBackoff: time.Millisecond,
		MaxBackoff:     5 * time.Millisecond,
		Multiplier:     2,
	})
	err := r.Execute(context.Background(), func(ctx context.Context) error {
		if calls.Add(1) < 3 {
			return errors.New("temp")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if calls.Load() != 3 {
		t.Fatalf("expected 3 calls, got %d", calls.Load())
	}
}

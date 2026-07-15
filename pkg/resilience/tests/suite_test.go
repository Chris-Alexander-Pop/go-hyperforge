package resilience_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/resilience"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/test"
)

// RetrySuite migrates retry smoke tests onto pkg/test.Suite.
type RetrySuite struct {
	test.Suite
}

func (s *RetrySuite) TestRetrySuccess() {
	calls := 0
	err := resilience.Retry(s.Ctx, resilience.DefaultRetryConfig(), func(ctx context.Context) error {
		calls++
		if calls < 3 {
			return errors.New("temp fail")
		}
		return nil
	})
	s.NoError(err)
	s.Equal(3, calls)
}

func (s *RetrySuite) TestRetryMaxAttempts() {
	cfg := resilience.DefaultRetryConfig()
	cfg.MaxAttempts = 3
	cfg.InitialBackoff = time.Millisecond

	calls := 0
	failErr := errors.New("steady fail")
	err := resilience.Retry(s.Ctx, cfg, func(ctx context.Context) error {
		calls++
		return failErr
	})
	s.Equal(failErr, err)
	s.Equal(3, calls)
}

func (s *RetrySuite) TestRetryContextCancel() {
	ctx, cancel := context.WithCancel(s.Ctx)
	cancel()
	cfg := resilience.DefaultRetryConfig()
	cfg.InitialBackoff = 100 * time.Millisecond

	err := resilience.Retry(ctx, cfg, func(ctx context.Context) error {
		return errors.New("should act on context")
	})
	s.True(errors.Is(err, context.Canceled))
}

func TestRetrySuite(t *testing.T) {
	test.Run(t, new(RetrySuite))
}

// CircuitBreakerSuite migrates CB state-transition smoke tests onto pkg/test.Suite.
type CircuitBreakerSuite struct {
	test.Suite
}

func (s *CircuitBreakerSuite) TestStateTransitions() {
	cb := resilience.NewCircuitBreaker(resilience.CircuitBreakerConfig{
		Name:             "suite-cb",
		FailureThreshold: 2,
		SuccessThreshold: 1,
		Timeout:          100 * time.Millisecond,
	})
	failErr := errors.New("failure")

	s.Equal(resilience.StateClosed, cb.State())
	s.Error(cb.Execute(s.Ctx, func(ctx context.Context) error { return failErr }))
	s.Equal(resilience.StateClosed, cb.State())
	s.Error(cb.Execute(s.Ctx, func(ctx context.Context) error { return failErr }))
	s.Equal(resilience.StateOpen, cb.State())

	err := cb.Execute(s.Ctx, func(ctx context.Context) error { return nil })
	s.True(errors.Is(err, resilience.ErrCircuitOpen))

	time.Sleep(150 * time.Millisecond)
	s.NoError(cb.Execute(s.Ctx, func(ctx context.Context) error { return nil }))
	s.Equal(resilience.StateClosed, cb.State())
}

func TestCircuitBreakerSuite(t *testing.T) {
	test.Run(t, new(CircuitBreakerSuite))
}

package tests

import (
	"errors"
	"testing"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/servicemesh/circuitbreaker"
	"github.com/stretchr/testify/suite"
)

// CircuitBreakerSuite provides tests for CircuitBreaker.
type CircuitBreakerSuite struct {
	suite.Suite
}

func (s *CircuitBreakerSuite) TestInitialStateClosed() {
	cb := circuitbreaker.New("test", circuitbreaker.Options{})
	s.Equal(circuitbreaker.StateClosed, cb.State())
}

func (s *CircuitBreakerSuite) TestSuccessfulExecution() {
	cb := circuitbreaker.New("test", circuitbreaker.Options{})

	result, err := cb.Execute(func() (interface{}, error) {
		return "success", nil
	})

	s.NoError(err)
	s.Equal("success", result)
	s.Equal(circuitbreaker.StateClosed, cb.State())
}

func (s *CircuitBreakerSuite) TestOpensAfterFailureThreshold() {
	cb := circuitbreaker.New("test", circuitbreaker.Options{
		FailureThreshold: 3,
	})

	testErr := errors.New("failure")

	// Fail 3 times
	for i := 0; i < 3; i++ {
		_, err := cb.Execute(func() (interface{}, error) {
			return nil, testErr
		})
		s.Error(err)
	}

	s.Equal(circuitbreaker.StateOpen, cb.State())
}

func (s *CircuitBreakerSuite) TestOpenCircuitRejectsRequests() {
	cb := circuitbreaker.New("test", circuitbreaker.Options{
		FailureThreshold: 1,
		Timeout:          10 * time.Second,
	})

	// Open the circuit
	cb.Execute(func() (interface{}, error) {
		return nil, errors.New("failure")
	})

	// Next request should be rejected immediately
	_, err := cb.Execute(func() (interface{}, error) {
		return "should not run", nil
	})

	s.Error(err)
	s.Equal(circuitbreaker.ErrCircuitOpen, err)
}

func (s *CircuitBreakerSuite) TestHalfOpenAfterTimeout() {
	cb := circuitbreaker.New("test", circuitbreaker.Options{
		FailureThreshold: 1,
		Timeout:          50 * time.Millisecond,
	})

	// Open the circuit
	cb.Execute(func() (interface{}, error) {
		return nil, errors.New("failure")
	})
	s.Equal(circuitbreaker.StateOpen, cb.State())

	// Wait for timeout
	time.Sleep(60 * time.Millisecond)

	// Next request should go through (half-open)
	_, err := cb.Execute(func() (interface{}, error) {
		return "success", nil
	})
	s.NoError(err)
}

func (s *CircuitBreakerSuite) TestClosesAfterSuccessThreshold() {
	cb := circuitbreaker.New("test", circuitbreaker.Options{
		FailureThreshold: 1,
		SuccessThreshold: 2,
		Timeout:          10 * time.Millisecond,
		MaxRequests:      5,
	})

	// Open the circuit
	cb.Execute(func() (interface{}, error) {
		return nil, errors.New("failure")
	})

	// Wait for timeout to transition to half-open
	time.Sleep(20 * time.Millisecond)

	// Succeed twice to close
	for i := 0; i < 2; i++ {
		cb.Execute(func() (interface{}, error) {
			return "success", nil
		})
	}

	s.Equal(circuitbreaker.StateClosed, cb.State())
}

func (s *CircuitBreakerSuite) TestReopensOnHalfOpenFailure() {
	cb := circuitbreaker.New("test", circuitbreaker.Options{
		FailureThreshold: 1,
		Timeout:          10 * time.Millisecond,
	})

	// Open the circuit
	cb.Execute(func() (interface{}, error) {
		return nil, errors.New("failure")
	})

	// Wait for timeout
	time.Sleep(20 * time.Millisecond)

	// Fail in half-open state
	cb.Execute(func() (interface{}, error) {
		return nil, errors.New("failure again")
	})

	s.Equal(circuitbreaker.StateOpen, cb.State())
}

func (s *CircuitBreakerSuite) TestSuccessResetsFailureCount() {
	cb := circuitbreaker.New("test", circuitbreaker.Options{
		FailureThreshold: 3,
	})

	// Fail twice
	for i := 0; i < 2; i++ {
		cb.Execute(func() (interface{}, error) {
			return nil, errors.New("failure")
		})
	}

	// Succeed once
	cb.Execute(func() (interface{}, error) {
		return "success", nil
	})

	// Fail twice more - should not open (count was reset)
	for i := 0; i < 2; i++ {
		cb.Execute(func() (interface{}, error) {
			return nil, errors.New("failure")
		})
	}

	s.Equal(circuitbreaker.StateClosed, cb.State())
}

func (s *CircuitBreakerSuite) TestForceOpen() {
	cb := circuitbreaker.New("test", circuitbreaker.Options{})
	s.Equal(circuitbreaker.StateClosed, cb.State())

	cb.ForceOpen()
	s.Equal(circuitbreaker.StateOpen, cb.State())
}

func (s *CircuitBreakerSuite) TestForceClose() {
	cb := circuitbreaker.New("test", circuitbreaker.Options{FailureThreshold: 1})

	cb.Execute(func() (interface{}, error) {
		return nil, errors.New("failure")
	})
	s.Equal(circuitbreaker.StateOpen, cb.State())

	cb.ForceClose()
	s.Equal(circuitbreaker.StateClosed, cb.State())
}

func (s *CircuitBreakerSuite) TestMetrics() {
	cb := circuitbreaker.New("test", circuitbreaker.Options{FailureThreshold: 5})

	for i := 0; i < 3; i++ {
		cb.Execute(func() (interface{}, error) {
			return nil, errors.New("failure")
		})
	}

	metrics := cb.Metrics()
	s.Equal(circuitbreaker.StateClosed, metrics.State)
	s.Equal(3, metrics.Failures)
}

func (s *CircuitBreakerSuite) TestOnStateChange() {
	var changes []circuitbreaker.State

	cb := circuitbreaker.New("test", circuitbreaker.Options{
		FailureThreshold: 1,
		OnStateChange: func(from, to circuitbreaker.State) {
			changes = append(changes, to)
		},
	})

	cb.Execute(func() (interface{}, error) {
		return nil, errors.New("failure")
	})

	// Give callback time to execute
	time.Sleep(10 * time.Millisecond)

	s.Contains(changes, circuitbreaker.StateOpen)
}

// TestCircuitBreakerSuite runs the test suite.
func TestCircuitBreakerSuite(t *testing.T) {
	suite.Run(t, new(CircuitBreakerSuite))
}

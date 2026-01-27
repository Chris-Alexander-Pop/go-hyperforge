package tests

import (
	"context"
	"testing"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/servicemesh/ratelimit"
	"github.com/stretchr/testify/suite"
)

// RateLimitSuite provides tests for rate limiters.
type RateLimitSuite struct {
	suite.Suite
}

func (s *RateLimitSuite) TestTokenBucketAllow() {
	limiter := ratelimit.NewTokenBucket(10, 10) // 10 capacity, 10/sec

	// Should allow first 10 requests
	for i := 0; i < 10; i++ {
		s.True(limiter.Allow())
	}

	// 11th should fail
	s.False(limiter.Allow())
}

func (s *RateLimitSuite) TestTokenBucketRefill() {
	limiter := ratelimit.NewTokenBucket(10, 100) // 10 capacity, 100/sec

	// Drain tokens
	for i := 0; i < 10; i++ {
		limiter.Allow()
	}
	s.False(limiter.Allow())

	// Wait for refill
	time.Sleep(50 * time.Millisecond) // Should add ~5 tokens

	s.True(limiter.Allow())
}

func (s *RateLimitSuite) TestTokenBucketAllowN() {
	limiter := ratelimit.NewTokenBucket(10, 10)

	s.True(limiter.AllowN(5))
	s.True(limiter.AllowN(5))
	s.False(limiter.AllowN(1))
}

func (s *RateLimitSuite) TestTokenBucketTokens() {
	limiter := ratelimit.NewTokenBucket(10, 10)

	s.InDelta(10.0, limiter.Tokens(), 0.1)

	limiter.AllowN(3)
	s.InDelta(7.0, limiter.Tokens(), 0.1)
}

func (s *RateLimitSuite) TestTokenBucketWait() {
	limiter := ratelimit.NewTokenBucket(1, 100) // 1 capacity, 100/sec
	limiter.Allow()                             // Drain

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := limiter.Wait(ctx)
	s.NoError(err)
}

func (s *RateLimitSuite) TestTokenBucketWaitTimeout() {
	limiter := ratelimit.NewTokenBucket(1, 0.1) // Very slow refill
	limiter.Allow()                             // Drain

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	err := limiter.Wait(ctx)
	s.Error(err)
}

func (s *RateLimitSuite) TestSlidingWindowAllow() {
	limiter := ratelimit.NewSlidingWindow(10, time.Second)

	for i := 0; i < 10; i++ {
		s.True(limiter.Allow())
	}

	s.False(limiter.Allow())
}

func (s *RateLimitSuite) TestSlidingWindowExpiry() {
	limiter := ratelimit.NewSlidingWindow(5, 50*time.Millisecond)

	for i := 0; i < 5; i++ {
		limiter.Allow()
	}
	s.False(limiter.Allow())

	// Wait for window to slide
	time.Sleep(60 * time.Millisecond)

	s.True(limiter.Allow())
}

func (s *RateLimitSuite) TestSlidingWindowTokens() {
	limiter := ratelimit.NewSlidingWindow(10, time.Second)

	s.Equal(10.0, limiter.Tokens())

	limiter.AllowN(3)
	s.Equal(7.0, limiter.Tokens())
}

func (s *RateLimitSuite) TestKeyedLimiter() {
	kl := ratelimit.NewKeyedLimiter(func() ratelimit.Limiter {
		return ratelimit.NewTokenBucket(2, 10)
	})

	// User A gets their own limit
	s.True(kl.Allow("user-a"))
	s.True(kl.Allow("user-a"))
	s.False(kl.Allow("user-a"))

	// User B has fresh limit
	s.True(kl.Allow("user-b"))
	s.True(kl.Allow("user-b"))
	s.False(kl.Allow("user-b"))
}

func (s *RateLimitSuite) TestKeyedLimiterAllowN() {
	kl := ratelimit.NewKeyedLimiter(func() ratelimit.Limiter {
		return ratelimit.NewTokenBucket(10, 10)
	})

	s.True(kl.AllowN("api-key-1", 5))
	s.True(kl.AllowN("api-key-1", 5))
	s.False(kl.AllowN("api-key-1", 1))
}

func (s *RateLimitSuite) TestKeyedLimiterWait() {
	kl := ratelimit.NewKeyedLimiter(func() ratelimit.Limiter {
		return ratelimit.NewTokenBucket(1, 100)
	})

	kl.Allow("key")

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := kl.Wait(ctx, "key")
	s.NoError(err)
}

// TestRateLimitSuite runs the test suite.
func TestRateLimitSuite(t *testing.T) {
	suite.Run(t, new(RateLimitSuite))
}

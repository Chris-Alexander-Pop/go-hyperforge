// Package ratelimit is a mesh-facing facade over pkg/algorithms/ratelimit.
//
// Prefer github.com/chris-alexander-pop/go-hyperforge/pkg/algorithms/ratelimit
// (tokenbucket / slidingwindow Local limiters) for new code. This package adapts
// those implementations to a simple Limiter interface used by mesh integrations.
//
// Usage:
//
//	limiter := ratelimit.NewTokenBucket(100, 10) // 100 tokens, 10/sec refill
//	if limiter.Allow() {
//	    // Process request
//	}
package ratelimit

import (
	"context"
	"sync"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/algorithms/ratelimit/slidingwindow"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/algorithms/ratelimit/tokenbucket"
)

// Algorithm represents the rate limiting algorithm.
type Algorithm string

const (
	AlgorithmTokenBucket   Algorithm = "token-bucket"
	AlgorithmSlidingWindow Algorithm = "sliding-window"
	AlgorithmFixedWindow   Algorithm = "fixed-window"
	AlgorithmLeakyBucket   Algorithm = "leaky-bucket"
)

// Config holds rate limiter configuration.
type Config struct {
	// Algorithm is the rate limiting algorithm.
	Algorithm Algorithm

	// Rate is requests per second.
	Rate float64

	// Burst is the maximum burst size.
	Burst int

	// Window is the time window for window-based algorithms.
	Window time.Duration
}

// Limiter defines the interface for rate limiters.
type Limiter interface {
	// Allow returns true if the request is allowed.
	Allow() bool

	// AllowN returns true if n tokens are available.
	AllowN(n int) bool

	// Wait blocks until a token is available or context is cancelled.
	Wait(ctx context.Context) error

	// Reserve reserves a token and returns the wait time.
	Reserve() time.Duration

	// Tokens returns the current number of available tokens.
	Tokens() float64
}

// TokenBucket adapts algorithms/ratelimit/tokenbucket.Local to Limiter.
type TokenBucket struct {
	inner *tokenbucket.Local
}

// NewTokenBucket creates a token-bucket limiter backed by pkg/algorithms.
func NewTokenBucket(capacity int, rate float64) *TokenBucket {
	return &TokenBucket{inner: tokenbucket.NewLocal(capacity, rate)}
}

func (tb *TokenBucket) Allow() bool                    { return tb.inner.Allow() }
func (tb *TokenBucket) AllowN(n int) bool              { return tb.inner.AllowN(n) }
func (tb *TokenBucket) Wait(ctx context.Context) error { return tb.inner.Wait(ctx) }
func (tb *TokenBucket) Reserve() time.Duration         { return tb.inner.Reserve() }
func (tb *TokenBucket) Tokens() float64                { return tb.inner.Tokens() }

// SlidingWindow adapts algorithms/ratelimit/slidingwindow.Local to Limiter.
type SlidingWindow struct {
	inner *slidingwindow.Local
}

// NewSlidingWindow creates a sliding-window limiter backed by pkg/algorithms.
func NewSlidingWindow(limit int, window time.Duration) *SlidingWindow {
	return &SlidingWindow{inner: slidingwindow.NewLocal(limit, window)}
}

func (sw *SlidingWindow) Allow() bool                    { return sw.inner.Allow() }
func (sw *SlidingWindow) AllowN(n int) bool              { return sw.inner.AllowN(n) }
func (sw *SlidingWindow) Wait(ctx context.Context) error { return sw.inner.Wait(ctx) }
func (sw *SlidingWindow) Reserve() time.Duration         { return sw.inner.Reserve() }
func (sw *SlidingWindow) Tokens() float64                { return sw.inner.Tokens() }

// KeyedLimiter provides per-key rate limiting.
type KeyedLimiter struct {
	mu       sync.RWMutex
	limiters map[string]Limiter
	factory  func() Limiter
}

// NewKeyedLimiter creates a new keyed limiter.
func NewKeyedLimiter(factory func() Limiter) *KeyedLimiter {
	return &KeyedLimiter{
		limiters: make(map[string]Limiter),
		factory:  factory,
	}
}

// Allow checks if a request for the key is allowed.
func (kl *KeyedLimiter) Allow(key string) bool {
	return kl.getLimiter(key).Allow()
}

// AllowN checks if n requests for the key are allowed.
func (kl *KeyedLimiter) AllowN(key string, n int) bool {
	return kl.getLimiter(key).AllowN(n)
}

// Wait blocks until a request for the key is allowed.
func (kl *KeyedLimiter) Wait(ctx context.Context, key string) error {
	return kl.getLimiter(key).Wait(ctx)
}

func (kl *KeyedLimiter) getLimiter(key string) Limiter {
	kl.mu.RLock()
	limiter, ok := kl.limiters[key]
	kl.mu.RUnlock()

	if ok {
		return limiter
	}

	kl.mu.Lock()
	defer kl.mu.Unlock()

	if limiter, ok = kl.limiters[key]; ok {
		return limiter
	}

	limiter = kl.factory()
	kl.limiters[key] = limiter
	return limiter
}

var (
	_ Limiter = (*TokenBucket)(nil)
	_ Limiter = (*SlidingWindow)(nil)
)

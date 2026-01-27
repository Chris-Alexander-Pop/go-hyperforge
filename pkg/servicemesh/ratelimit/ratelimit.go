// Package ratelimit provides rate limiting implementations.
//
// Supports multiple algorithms:
//   - Token Bucket: Allows bursts up to bucket capacity
//   - Sliding Window: Counts requests in a sliding time window
//   - Fixed Window: Counts requests in fixed time intervals
//   - Leaky Bucket: Smooths out request rate
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

// TokenBucket implements the token bucket algorithm.
type TokenBucket struct {
	mu         sync.Mutex
	tokens     float64
	capacity   float64
	rate       float64 // tokens per second
	lastUpdate time.Time
}

// NewTokenBucket creates a new token bucket limiter.
func NewTokenBucket(capacity int, rate float64) *TokenBucket {
	return &TokenBucket{
		tokens:     float64(capacity),
		capacity:   float64(capacity),
		rate:       rate,
		lastUpdate: time.Now(),
	}
}

func (tb *TokenBucket) refill() {
	now := time.Now()
	elapsed := now.Sub(tb.lastUpdate).Seconds()
	tb.tokens += elapsed * tb.rate
	if tb.tokens > tb.capacity {
		tb.tokens = tb.capacity
	}
	tb.lastUpdate = now
}

func (tb *TokenBucket) Allow() bool {
	return tb.AllowN(1)
}

func (tb *TokenBucket) AllowN(n int) bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.refill()

	if tb.tokens >= float64(n) {
		tb.tokens -= float64(n)
		return true
	}
	return false
}

func (tb *TokenBucket) Wait(ctx context.Context) error {
	for {
		if tb.Allow() {
			return nil
		}

		wait := tb.Reserve()
		if wait <= 0 {
			wait = time.Millisecond * 10
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wait):
		}
	}
}

func (tb *TokenBucket) Reserve() time.Duration {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.refill()

	if tb.tokens >= 1 {
		return 0
	}

	needed := 1 - tb.tokens
	return time.Duration(needed/tb.rate*1000) * time.Millisecond
}

func (tb *TokenBucket) Tokens() float64 {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	tb.refill()
	return tb.tokens
}

// SlidingWindow implements the sliding window algorithm.
type SlidingWindow struct {
	mu       sync.Mutex
	requests []time.Time
	limit    int
	window   time.Duration
}

// NewSlidingWindow creates a new sliding window limiter.
func NewSlidingWindow(limit int, window time.Duration) *SlidingWindow {
	return &SlidingWindow{
		requests: make([]time.Time, 0),
		limit:    limit,
		window:   window,
	}
}

func (sw *SlidingWindow) cleanup() {
	threshold := time.Now().Add(-sw.window)
	valid := make([]time.Time, 0, len(sw.requests))
	for _, t := range sw.requests {
		if t.After(threshold) {
			valid = append(valid, t)
		}
	}
	sw.requests = valid
}

func (sw *SlidingWindow) Allow() bool {
	return sw.AllowN(1)
}

func (sw *SlidingWindow) AllowN(n int) bool {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	sw.cleanup()

	if len(sw.requests)+n <= sw.limit {
		now := time.Now()
		for i := 0; i < n; i++ {
			sw.requests = append(sw.requests, now)
		}
		return true
	}
	return false
}

func (sw *SlidingWindow) Wait(ctx context.Context) error {
	for {
		if sw.Allow() {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Millisecond * 100):
		}
	}
}

func (sw *SlidingWindow) Reserve() time.Duration {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	sw.cleanup()

	if len(sw.requests) < sw.limit {
		return 0
	}

	// Wait until oldest request expires
	return time.Until(sw.requests[0].Add(sw.window))
}

func (sw *SlidingWindow) Tokens() float64 {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	sw.cleanup()
	return float64(sw.limit - len(sw.requests))
}

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

	// Double-check
	if limiter, ok = kl.limiters[key]; ok {
		return limiter
	}

	limiter = kl.factory()
	kl.limiters[key] = limiter
	return limiter
}

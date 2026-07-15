// Package ratelimit provides rate limiting algorithm implementations.
//
// This package contains the core data structures and algorithms for rate limiting:
//
//   - tokenbucket: Token bucket algorithm for smooth rate limiting
//   - leakybucket: Leaky bucket algorithm for traffic shaping
//   - fixedwindow: Fixed window counter algorithm
//   - slidingwindow: Sliding window log/counter algorithm
//   - htb: Hierarchical token bucket
//   - shaper: Traffic shaping utilities
//
// These algorithms are used by pkg/api/ratelimit for API-level rate limiting
// middleware and distributed rate limiting strategies.
//
// See also: github.com/chris-alexander-pop/go-hyperforge/pkg/api/ratelimit
package ratelimit

// Package ratelimit is a mesh-facing facade over pkg/algorithms/ratelimit.
//
// Prefer tokenbucket.Local / slidingwindow.Local (and related algorithms) for
// new rate-limit logic. This package adapts those implementations to the
// mesh Limiter interface.
package ratelimit

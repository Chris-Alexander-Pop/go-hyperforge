// Package circuitbreaker is a mesh-facing facade over pkg/resilience.
//
// Prefer pkg/resilience for application-level circuit breaking. This package
// preserves historical Options and Execute signatures for service-mesh code.
package circuitbreaker

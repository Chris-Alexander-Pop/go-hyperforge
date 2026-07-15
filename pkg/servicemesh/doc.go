/*
Package servicemesh provides service mesh components for microservices.

Subpackages:

  - circuitbreaker: Mesh-facing facade over pkg/resilience
  - discovery: Service discovery and registration
  - ratelimit: Mesh-facing facade over pkg/algorithms/ratelimit

Prefer pkg/resilience and pkg/algorithms/ratelimit for application code.
This package is a mesh-facing facade that preserves historical APIs while
delegating core behavior to those shared packages.

Usage:

	import "github.com/chris-alexander-pop/system-design-library/pkg/servicemesh/discovery"

	registry := consul.New(cfg)
	err := registry.Register(ctx, discovery.RegisterOptions{Name: "api", Port: 8080})
*/
package servicemesh

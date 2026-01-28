/*
Package servicemesh provides service mesh components for microservices.

Subpackages:

  - circuitbreaker: Circuit breaker pattern implementation
  - discovery: Service discovery and registration
  - ratelimit: Rate limiting algorithms

Usage:

	import "github.com/chris-alexander-pop/system-design-library/pkg/servicemesh/discovery"

	registry := consul.New(cfg)
	err := registry.Register(ctx, discovery.RegisterOptions{Name: "api", Port: 8080})
*/
package servicemesh

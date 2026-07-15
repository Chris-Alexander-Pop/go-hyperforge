/*
Package servicemesh provides mesh-facing facades and helpers for microservices.

# What this package is

Thin adapters and config helpers for service discovery, circuit breaking, and
rate limiting — plus optional mTLS transport helpers for discovery HTTP clients.

# What this package is not

It is not a full service mesh control/data plane. There is no sidecar proxy,
no automatic mTLS between workloads, no traffic shifting, and no mesh policy
API. Prefer a real mesh (Istio, Linkerd, Consul Connect) for those concerns.

# Subpackages

  - circuitbreaker: facade over pkg/resilience (prefer resilience directly)
  - discovery: ServiceRegistry + memory/Consul adapters; etcd/K8s still open
  - ratelimit: facade over pkg/algorithms/ratelimit

# Resilience & retry

Discovery and mesh client I/O should wrap calls with pkg/resilience.Retry or
NewRetrier for transient failures. Circuit breaker and rate-limit packages here
already delegate to shared implementations — do not reimplement backoff locally.

# mTLS

Root MTLSConfig + DialTLS / HTTPClient configure optional client certificates
for discovery adapters (see discovery.WithMTLS). Enabling MESH_MTLS_* env vars
only affects clients that opt in; it does not inject mesh-wide identity.

Usage:

	import "github.com/chris-alexander-pop/go-hyperforge/pkg/servicemesh/discovery"

	registry := consul.New(cfg)
	err := registry.Register(ctx, discovery.RegisterOptions{Name: "api", Port: 8080})
*/
package servicemesh

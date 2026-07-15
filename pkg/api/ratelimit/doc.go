/*
Package ratelimit provides API-level rate limiting middleware and strategies.

This package builds on the algorithm implementations in pkg/algorithms/ratelimit
to provide high-level rate limiting capabilities for APIs:

  - Distributed rate limiting with Redis/memory backends
  - Multiple strategy support (token bucket, sliding window, etc.)
  - Middleware integration for HTTP/gRPC

For the underlying algorithm implementations, see:
github.com/chris-alexander-pop/go-hyperforge/pkg/algorithms/ratelimit
*/
package ratelimit

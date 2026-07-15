/*
Package middleware provides HTTP middleware for auth context, RBAC RequirePermission,
rate limiting (IP / user / API-key keys), security headers, CORS, CSRF, circuit breaker,
cache, and audit.

gRPC interceptors live under pkg/api/grpc, not here. For Echo↔stdlib bridging, use
pkg/api/openapi (EchoMiddleware, EchoHandler, StdHandler, MountStd, ChainStd).
*/
package middleware

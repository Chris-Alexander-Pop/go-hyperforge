// Package grpc provides a gRPC server with OpenTelemetry, unary/stream recovery,
// AppError→gRPC status mapping (via pkg/errors.GRPCStatus) for unary and stream RPCs,
// AuthInterceptor / StreamAuthInterceptor (bearer metadata), reflection, and
// standard health checking (grpc.health.v1).
//
// Remaining gaps (not yet provided by this package):
//   - Per-method authorization / RBAC interceptors
//   - Stream-level logging interceptor
//   - Server TLS / mTLS configuration helpers
package grpc

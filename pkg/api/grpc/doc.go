// Package grpc provides a gRPC server with OpenTelemetry, unary/stream recovery,
// AppError→gRPC status mapping (via pkg/errors.GRPCStatus), reflection, and
// standard health checking (grpc.health.v1).
//
// Remaining gaps (not yet provided by this package):
//   - Auth / JWT unary+stream interceptors (use pkg/auth at the service layer)
//   - Per-method authorization / RBAC interceptors
//   - Stream-level logging and error-mapping interceptors (unary ErrorInterceptor only)
//   - Server TLS / mTLS configuration helpers
package grpc

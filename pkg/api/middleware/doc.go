/*
Package middleware provides HTTP middleware for auth context, RBAC RequirePermission,
rate limiting (IP / user / API-key keys), security headers, CORS, CSRF, circuit breaker,
cache, and audit.

gRPC interceptors live under pkg/api/grpc, not here. Echo-native middleware can wrap
stdlib handlers via echo.WrapMiddleware / echo.WrapHandler.
*/
package middleware

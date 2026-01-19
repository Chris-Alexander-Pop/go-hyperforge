/*
Package errors provides structured error handling for the system.

It defines a standard AppError type that includes:
  - Error Code (standardized strings like NOT_FOUND, INTERNAL)
  - Message (human-readable description)
  - Underlying Error (chaining)

It also provides helpers for common error scenarios and conversion to HTTP/gRPC status codes.
*/
package errors

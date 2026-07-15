/*
Package errors provides structured error handling with stable codes and protocol mapping.

It defines AppError with:
  - Code: standardized strings (NOT_FOUND, INTERNAL, DEADLINE_EXCEEDED, UNAVAILABLE, …)
  - Message: human-readable description
  - Err: underlying cause (unwrap-compatible)

Helpers (NotFound, InvalidArgument, DeadlineExceeded, Unavailable, ResourceExhausted,
Canceled, …) construct AppErrors with default messages when msg is empty.

Wrap preserves an AppError code when wrapping; IsCode and Code walk wraps via errors.As.
HTTPStatus and GRPCStatus map codes to HTTP (including 504/503/429/499) and gRPC statuses.
*/
package errors

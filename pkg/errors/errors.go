// Package errors provides structured error handling with error codes.
//
// This package provides:
//   - AppError: Structured error type with code, message, and wrapped error
//   - Standard error codes: NOT_FOUND, INVALID_ARGUMENT, INTERNAL, etc.
//   - Helper functions: NotFound, InvalidArgument, Internal, etc.
//   - Protocol conversion: HTTPStatus, GRPCStatus
//
// Usage:
//
//	import "github.com/chris-alexander-pop/system-design-library/pkg/errors"
//
//	// Create errors
//	err := errors.NotFound("user not found", nil)
//	err := errors.InvalidArgument("email is required", nil)
//	err := errors.Internal("database error", originalErr)
//
//	// Check and convert
//	status := errors.HTTPStatus(err)  // Returns HTTP status code
//	grpcStatus := errors.GRPCStatus(err)  // Returns gRPC status
package errors

import (
	"errors"
	"fmt"
	"net/http"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Standard error codes
const (
	CodeNotFound        = "NOT_FOUND"
	CodeInvalidArgument = "INVALID_ARGUMENT"
	CodeInternal        = "INTERNAL"
	CodeUnauthorized    = "UNAUTHORIZED"
	CodeForbidden       = "FORBIDDEN"
	CodeConflict        = "CONFLICT"

	// Aliases
	CodeUnauthenticated  = CodeUnauthorized
	CodePermissionDenied = CodeForbidden
)

// AppError is a custom error type that includes an error code, message, and underlying error.
type AppError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Err     error  `json:"-"`
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Err
}

// New creates a new AppError
func New(code string, message string, err error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// Helper functions for common errors

func NotFound(msg string, err error) *AppError {
	if msg == "" {
		msg = "resource not found"
	}
	return New(CodeNotFound, msg, err)
}

func InvalidArgument(msg string, err error) *AppError {
	if msg == "" {
		msg = "invalid argument"
	}
	return New(CodeInvalidArgument, msg, err)
}

func Internal(msg string, err error) *AppError {
	if msg == "" {
		msg = "internal server error"
	}
	return New(CodeInternal, msg, err)
}

func Unauthorized(msg string, err error) *AppError {
	if msg == "" {
		msg = "unauthorized"
	}
	return New(CodeUnauthorized, msg, err)
}

func Forbidden(msg string, err error) *AppError {
	if msg == "" {
		msg = "forbidden"
	}
	return New(CodeForbidden, msg, err)
}

func Conflict(msg string, err error) *AppError {
	if msg == "" {
		msg = "conflict"
	}
	return New(CodeConflict, msg, err)
}

// HTTPStatus returns the HTTP status code for a given error.
func HTTPStatus(err error) int {
	var appErr *AppError
	if errors.As(err, &appErr) {
		switch appErr.Code {
		case CodeNotFound:
			return http.StatusNotFound
		case CodeInvalidArgument:
			return http.StatusBadRequest
		case CodeUnauthorized:
			return http.StatusUnauthorized
		case CodeForbidden:
			return http.StatusForbidden
		case CodeConflict:
			return http.StatusConflict
		case CodeInternal:
			return http.StatusInternalServerError
		}
	}
	return http.StatusInternalServerError
}

// GRPCStatus returns the gRPC status for a given error.
func GRPCStatus(err error) *status.Status {
	var appErr *AppError
	if errors.As(err, &appErr) {
		switch appErr.Code {
		case CodeNotFound:
			return status.New(codes.NotFound, appErr.Message)
		case CodeInvalidArgument:
			return status.New(codes.InvalidArgument, appErr.Message)
		case CodeUnauthorized:
			return status.New(codes.Unauthenticated, appErr.Message)
		case CodeForbidden:
			return status.New(codes.PermissionDenied, appErr.Message)
		case CodeConflict:
			return status.New(codes.AlreadyExists, appErr.Message)
		case CodeInternal:
			return status.New(codes.Internal, appErr.Message)
		}
	}
	return status.New(codes.Unknown, err.Error())
}

// Wrap is a utility to wrap an error with a message
func Wrap(err error, msg string) error {
	return fmt.Errorf("%s: %w", msg, err)
}

// Is reports whether any error in err's chain matches target.
func Is(err, target error) bool {
	return errors.Is(err, target)
}

// As finds the first error in err's chain that matches target, and if so, sets target to that error value and returns true.
func As(err error, target any) bool {
	return errors.As(err, target)
}

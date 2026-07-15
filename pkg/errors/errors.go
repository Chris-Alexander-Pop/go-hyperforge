package errors

import (
	"errors"
	"fmt"
	"net/http"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Standard error codes.
const (
	CodeNotFound           = "NOT_FOUND"
	CodeInvalidArgument    = "INVALID_ARGUMENT"
	CodeInternal           = "INTERNAL"
	CodeUnauthorized       = "UNAUTHORIZED"
	CodeForbidden          = "FORBIDDEN"
	CodeConflict           = "CONFLICT"
	CodeUnimplemented      = "UNIMPLEMENTED"
	CodeDeadlineExceeded   = "DEADLINE_EXCEEDED"
	CodeUnavailable        = "UNAVAILABLE"
	CodeResourceExhausted  = "RESOURCE_EXHAUSTED"
	CodeCanceled           = "CANCELED"
	CodeAborted            = "ABORTED"
	CodeFailedPrecondition = "FAILED_PRECONDITION"

	// Aliases
	CodeUnauthenticated  = CodeUnauthorized
	CodePermissionDenied = CodeForbidden
)

// HTTP status for client-canceled requests (nginx / gRPC-gateway convention).
// Go's net/http has no named constant for 499.
const StatusClientClosedRequest = 499

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

// New creates a new AppError.
func New(code, message string, err error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// Helper functions for common errors

// NotFound returns an AppError with CodeNotFound.
func NotFound(msg string, err error) *AppError {
	if msg == "" {
		msg = "resource not found"
	}
	return New(CodeNotFound, msg, err)
}

// InvalidArgument returns an AppError with CodeInvalidArgument.
func InvalidArgument(msg string, err error) *AppError {
	if msg == "" {
		msg = "invalid argument"
	}
	return New(CodeInvalidArgument, msg, err)
}

// Internal returns an AppError with CodeInternal.
func Internal(msg string, err error) *AppError {
	if msg == "" {
		msg = "internal server error"
	}
	return New(CodeInternal, msg, err)
}

// Unauthorized returns an AppError with CodeUnauthorized.
func Unauthorized(msg string, err error) *AppError {
	if msg == "" {
		msg = "unauthorized"
	}
	return New(CodeUnauthorized, msg, err)
}

// Forbidden returns an AppError with CodeForbidden.
func Forbidden(msg string, err error) *AppError {
	if msg == "" {
		msg = "forbidden"
	}
	return New(CodeForbidden, msg, err)
}

// Conflict returns an AppError with CodeConflict.
func Conflict(msg string, err error) *AppError {
	if msg == "" {
		msg = "conflict"
	}
	return New(CodeConflict, msg, err)
}

// Unimplemented returns an AppError with CodeUnimplemented.
func Unimplemented(msg string, err error) *AppError {
	if msg == "" {
		msg = "not implemented"
	}
	return New(CodeUnimplemented, msg, err)
}

// DeadlineExceeded returns an AppError with CodeDeadlineExceeded.
func DeadlineExceeded(msg string, err error) *AppError {
	if msg == "" {
		msg = "deadline exceeded"
	}
	return New(CodeDeadlineExceeded, msg, err)
}

// Unavailable returns an AppError with CodeUnavailable.
func Unavailable(msg string, err error) *AppError {
	if msg == "" {
		msg = "service unavailable"
	}
	return New(CodeUnavailable, msg, err)
}

// ResourceExhausted returns an AppError with CodeResourceExhausted.
func ResourceExhausted(msg string, err error) *AppError {
	if msg == "" {
		msg = "resource exhausted"
	}
	return New(CodeResourceExhausted, msg, err)
}

// Canceled returns an AppError with CodeCanceled.
func Canceled(msg string, err error) *AppError {
	if msg == "" {
		msg = "canceled"
	}
	return New(CodeCanceled, msg, err)
}

// Aborted returns an AppError with CodeAborted.
func Aborted(msg string, err error) *AppError {
	if msg == "" {
		msg = "aborted"
	}
	return New(CodeAborted, msg, err)
}

// FailedPrecondition returns an AppError with CodeFailedPrecondition.
func FailedPrecondition(msg string, err error) *AppError {
	if msg == "" {
		msg = "failed precondition"
	}
	return New(CodeFailedPrecondition, msg, err)
}

// HTTPStatus returns the HTTP status code for a given error.
// If err unwraps to an AppError, the code is mapped; otherwise 500 is returned.
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
		case CodeUnimplemented:
			return http.StatusNotImplemented
		case CodeDeadlineExceeded:
			return http.StatusGatewayTimeout
		case CodeUnavailable:
			return http.StatusServiceUnavailable
		case CodeResourceExhausted:
			return http.StatusTooManyRequests
		case CodeCanceled:
			return StatusClientClosedRequest
		case CodeAborted:
			return http.StatusConflict
		case CodeFailedPrecondition:
			return http.StatusBadRequest
		}
	}
	return http.StatusInternalServerError
}

// GRPCStatus returns the gRPC status for a given error.
// If err unwraps to an AppError, the code is mapped; otherwise codes.Unknown is used.
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
		case CodeUnimplemented:
			return status.New(codes.Unimplemented, appErr.Message)
		case CodeDeadlineExceeded:
			return status.New(codes.DeadlineExceeded, appErr.Message)
		case CodeUnavailable:
			return status.New(codes.Unavailable, appErr.Message)
		case CodeResourceExhausted:
			return status.New(codes.ResourceExhausted, appErr.Message)
		case CodeCanceled:
			return status.New(codes.Canceled, appErr.Message)
		case CodeAborted:
			return status.New(codes.Aborted, appErr.Message)
		case CodeFailedPrecondition:
			return status.New(codes.FailedPrecondition, appErr.Message)
		}
	}
	if err == nil {
		return status.New(codes.OK, "")
	}
	return status.New(codes.Unknown, err.Error())
}

// FromHTTP maps an HTTP status code to an *AppError (message defaults by code).
func FromHTTP(statusCode int, msg string) *AppError {
	if msg == "" {
		msg = http.StatusText(statusCode)
	}
	switch statusCode {
	case http.StatusNotFound:
		return NotFound(msg, nil)
	case http.StatusBadRequest:
		return InvalidArgument(msg, nil)
	case http.StatusUnauthorized:
		return Unauthorized(msg, nil)
	case http.StatusForbidden:
		return Forbidden(msg, nil)
	case http.StatusConflict:
		return Conflict(msg, nil)
	case http.StatusNotImplemented:
		return Unimplemented(msg, nil)
	case http.StatusGatewayTimeout, http.StatusRequestTimeout:
		return DeadlineExceeded(msg, nil)
	case http.StatusServiceUnavailable:
		return Unavailable(msg, nil)
	case http.StatusTooManyRequests:
		return ResourceExhausted(msg, nil)
	case StatusClientClosedRequest:
		return Canceled(msg, nil)
	case http.StatusPreconditionFailed:
		return FailedPrecondition(msg, nil)
	default:
		if statusCode >= 500 {
			return Internal(msg, nil)
		}
		return InvalidArgument(msg, nil)
	}
}

// FromGRPC maps a gRPC code to an *AppError.
func FromGRPC(code codes.Code, msg string) *AppError {
	if msg == "" {
		msg = code.String()
	}
	switch code {
	case codes.NotFound:
		return NotFound(msg, nil)
	case codes.InvalidArgument:
		return InvalidArgument(msg, nil)
	case codes.Unauthenticated:
		return Unauthorized(msg, nil)
	case codes.PermissionDenied:
		return Forbidden(msg, nil)
	case codes.AlreadyExists:
		return Conflict(msg, nil)
	case codes.Unimplemented:
		return Unimplemented(msg, nil)
	case codes.DeadlineExceeded:
		return DeadlineExceeded(msg, nil)
	case codes.Unavailable:
		return Unavailable(msg, nil)
	case codes.ResourceExhausted:
		return ResourceExhausted(msg, nil)
	case codes.Canceled:
		return Canceled(msg, nil)
	case codes.Aborted:
		return Aborted(msg, nil)
	case codes.FailedPrecondition:
		return FailedPrecondition(msg, nil)
	case codes.Internal:
		return Internal(msg, nil)
	default:
		return Internal(msg, nil)
	}
}

// Wrap wraps err with msg. If err is or unwraps to an *AppError, the AppError
// code is preserved and an *AppError is returned with the wrapped cause.
// Otherwise a plain fmt.Errorf("%s: %w", msg, err) is returned.
func Wrap(err error, msg string) error {
	if err == nil {
		return nil
	}
	var appErr *AppError
	if errors.As(err, &appErr) {
		return New(appErr.Code, msg, err)
	}
	return fmt.Errorf("%s: %w", msg, err)
}

// IsCode reports whether err is or unwraps to an *AppError with the given code.
func IsCode(err error, code string) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code == code
	}
	return false
}

// Code returns the AppError code if err is or unwraps to an *AppError.
// Otherwise it returns an empty string.
func Code(err error) string {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code
	}
	return ""
}

// Is reports whether any error in err's chain matches target.
func Is(err, target error) bool {
	return errors.Is(err, target)
}

// As finds the first error in err's chain that matches target, and if so, sets
// target to that error value and returns true.
func As(err error, target any) bool {
	return errors.As(err, target)
}

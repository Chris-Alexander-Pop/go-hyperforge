package audit

import (
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
)

// Error codes for audit operations.
const (
	CodeInvalidArgument = "AUDIT_INVALID_ARGUMENT"
	CodeAppendFailed    = "AUDIT_APPEND_FAILED"
	CodeQueryFailed     = "AUDIT_QUERY_FAILED"
	CodeNotSupported    = "AUDIT_NOT_SUPPORTED"
	CodeMarshalFailed   = "AUDIT_MARSHAL_FAILED"
)

// ErrNotSupported is returned when an adapter does not support an operation
// (for example, Query on a stdout-only sink).
var ErrNotSupported = errors.New(CodeNotSupported, "operation not supported by this audit store", nil)

// ErrInvalidArgument returns an invalid-argument error for audit operations.
func ErrInvalidArgument(msg string, err error) *errors.AppError {
	if msg == "" {
		msg = "invalid argument"
	}
	return errors.New(CodeInvalidArgument, msg, err)
}

// ErrAppendFailed wraps a failure while persisting an audit event.
func ErrAppendFailed(msg string, err error) *errors.AppError {
	if msg == "" {
		msg = "failed to append audit event"
	}
	return errors.New(CodeAppendFailed, msg, err)
}

// ErrQueryFailed wraps a failure while querying audit events.
func ErrQueryFailed(msg string, err error) *errors.AppError {
	if msg == "" {
		msg = "failed to query audit events"
	}
	return errors.New(CodeQueryFailed, msg, err)
}

// ErrMarshalFailed wraps a failure encoding an audit event.
func ErrMarshalFailed(err error) *errors.AppError {
	return errors.New(CodeMarshalFailed, "failed to marshal audit event", err)
}

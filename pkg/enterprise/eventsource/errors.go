package eventsource

import (
	"fmt"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
)

// Error codes for event-sourcing operations.
const (
	CodeVersionConflict = "EVENTSOURCE_VERSION_CONFLICT"
	CodeInvalidArgument = "EVENTSOURCE_INVALID_ARGUMENT"
	CodeNotFound        = "EVENTSOURCE_NOT_FOUND"
	CodeApplyFailed     = "EVENTSOURCE_APPLY_FAILED"
)

// ErrVersionConflict is returned when Append's expectedVersion does not match
// the current stream version.
var ErrVersionConflict = errors.New(CodeVersionConflict, "event store version conflict", nil)

// VersionConflict returns a conflict error with aggregate and version detail.
func VersionConflict(aggregateID string, expected, actual int) *errors.AppError {
	msg := fmt.Sprintf("version conflict for aggregate %q: expected %d, actual %d", aggregateID, expected, actual)
	return errors.New(CodeVersionConflict, msg, nil)
}

// ErrInvalidArgument returns an invalid-argument error for eventsource operations.
func ErrInvalidArgument(msg string, err error) *errors.AppError {
	if msg == "" {
		msg = "invalid argument"
	}
	return errors.New(CodeInvalidArgument, msg, err)
}

// ErrAggregateNotFound returns a not-found error for a missing aggregate stream.
func ErrAggregateNotFound(aggregateID string) *errors.AppError {
	return errors.New(CodeNotFound, fmt.Sprintf("aggregate %q not found", aggregateID), nil)
}

// ErrApplyFailed returns an error when applying an event to an aggregate fails.
func ErrApplyFailed(msg string, err error) *errors.AppError {
	if msg == "" {
		msg = "failed to apply event"
	}
	return errors.New(CodeApplyFailed, msg, err)
}

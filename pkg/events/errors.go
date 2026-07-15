package events

import "github.com/chris-alexander-pop/system-design-library/pkg/errors"

// Error codes for event bus operations.
const (
	CodeClosed               = "EVENTS_CLOSED"
	CodeInvalidEvent         = "EVENTS_INVALID_EVENT"
	CodeInvalidTopic         = "EVENTS_INVALID_TOPIC"
	CodeHandlerFailed        = "EVENTS_HANDLER_FAILED"
	CodeSubscriptionNotFound = "EVENTS_SUBSCRIPTION_NOT_FOUND"
)

// ErrClosed creates an error when the bus is closed.
func ErrClosed(err error) *errors.AppError {
	return errors.New(CodeClosed, "event bus is closed", err)
}

// ErrInvalidEvent creates an error for a malformed event.
func ErrInvalidEvent(msg string, err error) *errors.AppError {
	if msg == "" {
		msg = "invalid event"
	}
	return errors.New(CodeInvalidEvent, msg, err)
}

// ErrInvalidTopic creates an error for an empty or invalid topic.
func ErrInvalidTopic(topic string, err error) *errors.AppError {
	return errors.New(CodeInvalidTopic, "invalid event topic: "+topic, err)
}

// ErrHandlerFailed creates an error when one or more handlers fail during Publish.
func ErrHandlerFailed(err error) *errors.AppError {
	return errors.New(CodeHandlerFailed, "one or more event handlers failed", err)
}

// ErrSubscriptionNotFound creates an error when Unsubscribe cannot find the ID.
func ErrSubscriptionNotFound(id string, err error) *errors.AppError {
	return errors.New(CodeSubscriptionNotFound, "subscription not found: "+id, err)
}

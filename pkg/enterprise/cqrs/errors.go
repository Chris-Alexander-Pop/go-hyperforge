package cqrs

import (
	"fmt"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
)

// Error codes for CQRS operations.
const (
	CodeHandlerNotFound = "CQRS_HANDLER_NOT_FOUND"
	CodeInvalidCommand  = "CQRS_INVALID_COMMAND"
	CodeInvalidQuery    = "CQRS_INVALID_QUERY"
)

// ErrCommandHandlerNotFound returns a not-found error for a missing command handler.
func ErrCommandHandlerNotFound(commandName string) *errors.AppError {
	return errors.New(CodeHandlerNotFound, fmt.Sprintf("no handler registered for command: %s", commandName), nil)
}

// ErrQueryHandlerNotFound returns a not-found error for a missing query handler.
func ErrQueryHandlerNotFound(queryName string) *errors.AppError {
	return errors.New(CodeHandlerNotFound, fmt.Sprintf("no handler registered for query: %s", queryName), nil)
}

// ErrInvalidCommand returns an invalid-argument error for a bad command.
func ErrInvalidCommand(msg string, err error) *errors.AppError {
	if msg == "" {
		msg = "invalid command"
	}
	return errors.New(CodeInvalidCommand, msg, err)
}

// ErrInvalidQuery returns an invalid-argument error for a bad query.
func ErrInvalidQuery(msg string, err error) *errors.AppError {
	if msg == "" {
		msg = "invalid query"
	}
	return errors.New(CodeInvalidQuery, msg, err)
}

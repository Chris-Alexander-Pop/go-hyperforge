package metering

import "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"

const (
	// CodeClosed indicates the meter or rater was used after Close.
	CodeClosed = "CLOSED"
)

var (
	// ErrRateNotFound is returned when pricing information is missing for a resource type.
	ErrRateNotFound = errors.NotFound("rate not found", nil)

	// ErrInvalidUsage is returned when usage data or a rate card is malformed.
	ErrInvalidUsage = errors.InvalidArgument("invalid usage data", nil)
)

// ErrClosed creates an error when the meter or rater is closed.
func ErrClosed(err error) *errors.AppError {
	return errors.New(CodeClosed, "metering is closed", err)
}

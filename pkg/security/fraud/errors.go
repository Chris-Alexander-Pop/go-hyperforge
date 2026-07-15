package fraud

import "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"

const (
	CodeInvalidEvent = "FRAUD_INVALID_EVENT"
	CodeScoreFailed  = "FRAUD_SCORE_FAILED"
	CodeUnavailable  = "FRAUD_UNAVAILABLE"
)

var (
	// ErrInvalidEvent is returned when a UserEvent is malformed.
	ErrInvalidEvent = errors.New(CodeInvalidEvent, "invalid fraud event", nil)

	// ErrScoreFailed is returned when scoring cannot be completed.
	ErrScoreFailed = errors.New(CodeScoreFailed, "fraud scoring failed", nil)

	// ErrUnavailable is returned when a remote fraud provider is unreachable.
	ErrUnavailable = errors.New(CodeUnavailable, "fraud provider unavailable", nil)
)

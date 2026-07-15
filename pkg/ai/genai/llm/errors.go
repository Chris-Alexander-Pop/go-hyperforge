package llm

import "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"

// Domain error sentinels for LLM clients.
var (
	// ErrEmptyMessages is returned when Chat/StreamChat receives no messages.
	ErrEmptyMessages = errors.InvalidArgument("messages are required", nil)

	// ErrNilClient is returned when a required client dependency is nil.
	ErrNilClient = errors.InvalidArgument("llm client is required", nil)

	// ErrEmptyContent is returned when a message has empty content where required.
	ErrEmptyContent = errors.InvalidArgument("message content is required", nil)

	// ErrProvider is a generic upstream provider failure.
	ErrProvider = errors.Unavailable("llm provider unavailable", nil)

	// ErrCanceled surfaces context cancellation as a typed domain error when wrapping.
	ErrCanceled = errors.Canceled("llm request canceled", nil)
)

// WrapProvider wraps an upstream provider error as UNAVAILABLE.
func WrapProvider(err error) error {
	if err == nil {
		return nil
	}
	return errors.Unavailable("llm provider error", err)
}

package inference

import "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"

// Sentinel errors for inference operations.
var (
	// ErrModelNotFound is returned when a model is not loaded.
	ErrModelNotFound = errors.NotFound("model not found", nil)

	// ErrModelAlreadyLoaded is returned when loading a duplicate model name.
	ErrModelAlreadyLoaded = errors.Conflict("model already loaded", nil)

	// ErrInvalidRequest is returned when a predict request is malformed.
	ErrInvalidRequest = errors.InvalidArgument("invalid inference request", nil)

	// ErrNotReady is returned when the model is not ready for inference.
	ErrNotReady = errors.Unavailable("model not ready", nil)

	// ErrClosed is returned when the server was used after Close.
	ErrClosed = errors.New("INFERENCE_CLOSED", "inference server is closed", nil)
)

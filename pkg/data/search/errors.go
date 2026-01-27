package search

import "github.com/chris-alexander-pop/system-design-library/pkg/errors"

// Sentinel errors for search operations.
var (
	// ErrIndexNotFound is returned when an index does not exist.
	ErrIndexNotFound = errors.NotFound("index not found", nil)

	// ErrIndexAlreadyExists is returned when trying to create an existing index.
	ErrIndexAlreadyExists = errors.Conflict("index already exists", nil)

	// ErrDocumentNotFound is returned when a document does not exist.
	ErrDocumentNotFound = errors.NotFound("document not found", nil)

	// ErrInvalidQuery is returned when the search query is malformed.
	ErrInvalidQuery = errors.InvalidArgument("invalid search query", nil)

	// ErrInvalidMapping is returned when the index mapping is invalid.
	ErrInvalidMapping = errors.InvalidArgument("invalid index mapping", nil)

	// ErrConnectionFailed is returned when the search backend is unreachable.
	ErrConnectionFailed = errors.Internal("search backend connection failed", nil)

	// ErrTimeout is returned when the operation times out.
	ErrTimeout = errors.Internal("search operation timed out", nil)

	// ErrBulkPartialFailure is returned when some bulk operations failed.
	ErrBulkPartialFailure = errors.Internal("some bulk operations failed", nil)
)

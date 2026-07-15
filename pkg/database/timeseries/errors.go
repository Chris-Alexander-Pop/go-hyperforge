package timeseries

import "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"

var (
	// ErrNotFound indicates that the requested data was not found.
	ErrNotFound = errors.NotFound

	// ErrInvalidQuery indicates that the query string is invalid.
	ErrInvalidQuery = errors.InvalidArgument

	// ErrConnectionFailed indicates a failure to connect to the database.
	ErrConnectionFailed = errors.Internal
)

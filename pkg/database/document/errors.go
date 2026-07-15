package document

import "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"

var (
	// ErrNotFound indicates that the requested document was not found.
	ErrNotFound = errors.NotFound

	// ErrAlreadyExists indicates that the document already exists.
	ErrAlreadyExists = errors.Conflict

	// ErrInvalidQuery indicates that the query or filter is invalid.
	ErrInvalidQuery = errors.InvalidArgument
)

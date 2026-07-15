package feature

import "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"

// Sentinel errors for feature store operations.
var (
	// ErrGroupNotFound is returned when a feature group does not exist.
	ErrGroupNotFound = errors.NotFound("feature group not found", nil)

	// ErrGroupExists is returned when creating a duplicate feature group.
	ErrGroupExists = errors.Conflict("feature group already exists", nil)

	// ErrInvalidGroup is returned when a feature group definition is invalid.
	ErrInvalidGroup = errors.InvalidArgument("invalid feature group", nil)

	// ErrInvalidVector is returned when feature vectors are malformed.
	ErrInvalidVector = errors.InvalidArgument("invalid feature vector", nil)
)

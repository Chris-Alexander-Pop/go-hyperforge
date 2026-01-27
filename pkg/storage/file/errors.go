package file

import "github.com/chris-alexander-pop/system-design-library/pkg/errors"

// Sentinel errors for file storage operations.
var (
	// ErrNotFound is returned when a file or directory does not exist.
	ErrNotFound = errors.NotFound("file not found", nil)

	// ErrAlreadyExists is returned when attempting to create a file that already exists.
	ErrAlreadyExists = errors.Conflict("file already exists", nil)

	// ErrIsDirectory is returned when a file operation is attempted on a directory.
	ErrIsDirectory = errors.InvalidArgument("path is a directory", nil)

	// ErrNotDirectory is returned when a directory operation is attempted on a file.
	ErrNotDirectory = errors.InvalidArgument("path is not a directory", nil)

	// ErrPermissionDenied is returned when the operation is not permitted.
	ErrPermissionDenied = errors.Forbidden("permission denied", nil)

	// ErrFileTooLarge is returned when a file exceeds the maximum allowed size.
	ErrFileTooLarge = errors.InvalidArgument("file too large", nil)

	// ErrInvalidPath is returned when the path is malformed.
	ErrInvalidPath = errors.InvalidArgument("invalid path", nil)
)

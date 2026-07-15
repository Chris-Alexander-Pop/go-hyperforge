package database

import "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"

// Sentinel errors for database operations.
var (
	// ErrConnectionFailed is returned when a database connection cannot be established.
	ErrConnectionFailed = errors.New(errors.CodeInternal, "database connection failed", nil)

	// ErrShardNotFound is returned when a shard cannot be resolved.
	ErrShardNotFound = errors.New(errors.CodeNotFound, "shard not found", nil)

	// ErrInvalidDriver is returned when an unsupported driver is specified.
	ErrInvalidDriver = errors.New(errors.CodeInvalidArgument, "invalid database driver", nil)

	// ErrInvalidStoreType is returned when an unsupported store type is specified.
	ErrInvalidStoreType = errors.New(errors.CodeInvalidArgument, "invalid store type", nil)

	// ErrNotImplemented is returned when an operation is not supported.
	ErrNotImplemented = errors.New(errors.CodeInternal, "operation not implemented", nil)
)

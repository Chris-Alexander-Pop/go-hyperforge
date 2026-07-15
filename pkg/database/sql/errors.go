package sql

import "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"

// Sentinel errors for SQL database operations.
var (
	// ErrConnectionFailed is returned when a database connection cannot be established.
	ErrConnectionFailed = errors.New(errors.CodeInternal, "database connection failed", nil)

	// ErrQueryFailed is returned when a query execution fails.
	ErrQueryFailed = errors.New(errors.CodeInternal, "query execution failed", nil)

	// ErrTransactionFailed is returned when a transaction cannot be started or committed.
	ErrTransactionFailed = errors.New(errors.CodeInternal, "transaction failed", nil)

	// ErrInvalidDriver is returned when an unsupported driver is specified.
	ErrInvalidDriver = errors.New(errors.CodeInvalidArgument, "invalid database driver", nil)

	// ErrShardNotFound is returned when a shard cannot be resolved.
	ErrShardNotFound = errors.New(errors.CodeNotFound, "shard not found", nil)
)

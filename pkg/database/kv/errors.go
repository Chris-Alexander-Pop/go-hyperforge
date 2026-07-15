package kv

import "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"

// Sentinel errors for key-value database operations.
var (
	// ErrKeyNotFound is returned when a key does not exist.
	ErrKeyNotFound = errors.New(errors.CodeNotFound, "key not found", nil)

	// ErrConnectionFailed is returned when a connection cannot be established.
	ErrConnectionFailed = errors.New(errors.CodeInternal, "kv connection failed", nil)

	// ErrInvalidDriver is returned when an unsupported driver is specified.
	ErrInvalidDriver = errors.New(errors.CodeInvalidArgument, "invalid kv driver", nil)
)

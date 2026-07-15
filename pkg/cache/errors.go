package cache

import "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"

// Sentinel errors for cache operations.
var (
	// ErrKeyNotFound is returned when a key does not exist in the cache.
	ErrKeyNotFound = errors.NotFound("key not found", nil)

	// ErrKeyExpired is returned when a key exists but has expired.
	ErrKeyExpired = errors.NotFound("key expired", nil)
)

// IsNotFound reports whether err is a cache miss (NOT_FOUND).
func IsNotFound(err error) bool {
	return errors.IsCode(err, errors.CodeNotFound)
}

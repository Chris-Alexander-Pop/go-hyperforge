// Package cache provides a unified caching interface with multiple backend support.
//
// This package supports the following backends:
//   - Memory: In-memory cache for testing and development
//   - Redis: Production-grade distributed cache
//
// Usage:
//
//	import "github.com/chris-alexander-pop/system-design-library/pkg/cache/adapters/memory"
//
//	cache := memory.New()
//	defer cache.Close()
//
//	err := cache.Set(ctx, "key", value, time.Hour)
//	err = cache.Get(ctx, "key", &result)
package cache

import (
	"context"
	"time"
)

// Cache defines the standard caching interface.
type Cache interface {
	// Get retrieves a value by key and unmarshals into dest.
	// Returns errors.NotFound if the key does not exist or has expired.
	Get(ctx context.Context, key string, dest interface{}) error

	// Set stores a value with a TTL.
	// A TTL of 0 means no expiration.
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error

	// Delete removes a key from the cache.
	// Returns nil if the key does not exist.
	Delete(ctx context.Context, key string) error

	// Incr increments a counter by delta and returns the new value.
	Incr(ctx context.Context, key string, delta int64) (int64, error)

	// Close releases all resources.
	Close() error
}

// Config holds configuration for the Cache.
type Config struct {
	// Driver specifies the cache backend: "memory" or "redis".
	Driver string `env:"CACHE_DRIVER" env-default:"memory"`

	// Host is the cache server hostname.
	Host string `env:"CACHE_HOST" env-default:"localhost"`

	// Port is the cache server port.
	Port string `env:"CACHE_PORT" env-default:"6379"`

	// Password is the authentication password (optional).
	Password string `env:"CACHE_PASSWORD"`

	// DB is the database number (Redis only).
	DB int `env:"CACHE_DB" env-default:"0"`
}

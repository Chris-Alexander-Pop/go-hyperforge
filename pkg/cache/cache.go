// Package cache provides a unified caching interface with multiple backend support.
//
// This package supports the following backends:
//   - Memory: In-memory cache for testing and development
//   - Redis: Production-grade distributed cache
//
// Usage:
//
//	import (
//		"github.com/chris-alexander-pop/go-hyperforge/pkg/cache"
//		_ "github.com/chris-alexander-pop/go-hyperforge/pkg/cache/adapters/memory"
//	)
//
//	c, err := cache.NewFromConfig(cache.Config{Driver: "memory"})
//	defer c.Close()
//
//	err = c.Set(ctx, "key", value, time.Hour)
//	err = c.Get(ctx, "key", &result)
//	ok, err := c.Exists(ctx, "key")
//	n, err := cache.InvalidatePrefix(ctx, c, "user:")
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

	// Exists reports whether key is present and not expired.
	Exists(ctx context.Context, key string) (bool, error)

	// MGet retrieves multiple keys. dest must be a non-nil map[string]T pointer
	// (e.g. *map[string]string). Missing keys are omitted from the map.
	MGet(ctx context.Context, keys []string, dest interface{}) error

	// MSet stores multiple key/value pairs with the same TTL.
	// A TTL of 0 means no expiration.
	MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error

	// Expire sets or clears the TTL on an existing key.
	// Returns ErrKeyNotFound if the key is missing or expired.
	// ttl <= 0 removes expiration (key persists).
	Expire(ctx context.Context, key string, ttl time.Duration) error

	// GetTTL returns the remaining TTL for key.
	// Returns -1 if the key exists with no expiration.
	// Returns ErrKeyNotFound if the key is missing or expired.
	GetTTL(ctx context.Context, key string) (time.Duration, error)

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

	// PoolSize is the Redis connection pool size (0 = client default).
	PoolSize int `env:"CACHE_POOL_SIZE" env-default:"0"`

	// DialTimeout is the Redis dial timeout (0 = client default).
	DialTimeout time.Duration `env:"CACHE_DIAL_TIMEOUT" env-default:"0"`

	// ReadTimeout is the Redis read timeout (0 = client default).
	ReadTimeout time.Duration `env:"CACHE_READ_TIMEOUT" env-default:"0"`

	// WriteTimeout is the Redis write timeout (0 = client default).
	WriteTimeout time.Duration `env:"CACHE_WRITE_TIMEOUT" env-default:"0"`

	// TLS enables TLS for Redis connections.
	TLS bool `env:"CACHE_TLS" env-default:"false"`

	// Cluster enables Redis Cluster mode. When true, DB is ignored and Addrs
	// (or Host:Port as a single seed) are used. MGet/MSet require same-slot keys.
	Cluster bool `env:"CACHE_CLUSTER" env-default:"false"`

	// Addrs are Redis Cluster seed addresses ("host:port"). Used when Cluster is true.
	// When empty and Cluster is true, Host:Port is used as the sole seed.
	Addrs []string `env:"CACHE_ADDRS"`
}

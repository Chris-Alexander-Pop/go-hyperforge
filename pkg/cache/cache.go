package cache

import (
	"context"
	"time"
)

// Cache defines the standard Caching interface
type Cache interface {
	// Get retrieves a value by key.
	// It should handle unmarshaling into 'dest' if possible, or return []byte.
	Get(ctx context.Context, key string, dest interface{}) error

	// Set stores a value with a TTL.
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error

	// Delete removes a key.
	Delete(ctx context.Context, key string) error

	// Close cleans up resources
	Close() error
}

// Config holds configuration for the Cache
type Config struct {
	Driver   string `env:"CACHE_DRIVER" env-default:"memory"` // memory, redis
	Host     string `env:"CACHE_HOST" env-default:"localhost"`
	Port     string `env:"CACHE_PORT" env-default:"6379"`
	Password string `env:"CACHE_PASSWORD"`
	DB       int    `env:"CACHE_DB" env-default:"0"`
}

// Package kv provides a unified interface for key-value databases.
//
// Supported backends:
//   - Redis: Production-grade in-memory key-value store
//   - Cassandra: Distributed NoSQL database
//   - Memory: In-memory store for testing
//
// Usage:
//
//	import "github.com/chris-alexander-pop/go-hyperforge/pkg/database/kv/adapters/redis"
//
//	client, err := redis.New(cfg)
//	err = client.Set(ctx, "key", []byte("value"), time.Hour)
package kv

import (
	"context"
	"time"
)

// Config holds configuration for a key-value database.
type Config struct {
	// Driver specifies the KV backend: "redis", "cassandra", "memory".
	Driver string `env:"KV_DRIVER" env-default:"redis"`

	// Host is the database server hostname.
	Host string `env:"KV_HOST" env-default:"localhost"`

	// Port is the database server port.
	Port string `env:"KV_PORT" env-default:"6379"`

	// Password is the authentication password.
	Password string `env:"KV_PASSWORD"`

	// Database is the database number (Redis-specific).
	Database int `env:"KV_DATABASE" env-default:"0"`

	// SSL Configuration
	SSLMode string `env:"KV_SSL_MODE" env-default:"disable"`

	// Connection Pooling
	PoolSize     int `env:"KV_POOL_SIZE" env-default:"10"`
	MinIdleConns int `env:"KV_MIN_IDLE_CONNS" env-default:"5"`
}

// KV defines the interface for key-value database operations.
type KV interface {
	// Get retrieves a value by key.
	// Returns errors.NotFound if the key does not exist.
	Get(ctx context.Context, key string) ([]byte, error)

	// Set stores a value with the given TTL.
	// A TTL of 0 means no expiration.
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error

	// Delete removes a key.
	// Returns nil if the key does not exist.
	Delete(ctx context.Context, key string) error

	// Exists checks if a key exists.
	Exists(ctx context.Context, key string) (bool, error)

	// Close releases all resources.
	Close() error
}

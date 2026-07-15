package redis

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/database/kv"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/redis/go-redis/v9"
)

// Adapter implements kv.KV for Redis.
type Adapter struct {
	client *redis.Client
}

// New creates a new Redis adapter.
func New(cfg kv.Config) (*Adapter, error) {
	opts := &redis.Options{
		Addr:         fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
		Password:     cfg.Password,
		DB:           cfg.Database,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
	}

	// SSL Support
	if cfg.SSLMode == "require" || cfg.SSLMode == "true" {
		opts.TLSConfig = &tls.Config{
			InsecureSkipVerify: cfg.SSLMode != "verify-full",
		}
	}

	client := redis.NewClient(opts)

	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, errors.Wrap(err, "failed to ping redis")
	}

	return &Adapter{client: client}, nil
}

// Get retrieves a value by key.
func (a *Adapter) Get(ctx context.Context, key string) ([]byte, error) {
	val, err := a.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, errors.NotFound("key not found", nil)
	}
	if err != nil {
		return nil, errors.Internal("redis get failed", err)
	}
	return val, nil
}

// Set stores a value with the given TTL.
func (a *Adapter) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	err := a.client.Set(ctx, key, value, ttl).Err()
	if err != nil {
		return errors.Internal("redis set failed", err)
	}
	return nil
}

// Delete removes a key.
func (a *Adapter) Delete(ctx context.Context, key string) error {
	err := a.client.Del(ctx, key).Err()
	if err != nil {
		return errors.Internal("redis delete failed", err)
	}
	return nil
}

// Exists checks if a key exists.
func (a *Adapter) Exists(ctx context.Context, key string) (bool, error) {
	n, err := a.client.Exists(ctx, key).Result()
	if err != nil {
		return false, errors.Internal("redis exists failed", err)
	}
	return n > 0, nil
}

// Close closes the Redis connection.
func (a *Adapter) Close() error {
	return a.client.Close()
}

// Client returns the underlying Redis client for advanced operations.
func (a *Adapter) Client() *redis.Client {
	return a.client
}

// Ensure Adapter implements kv.KV
var _ kv.KV = (*Adapter)(nil)

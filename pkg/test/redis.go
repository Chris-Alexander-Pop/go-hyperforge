package test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go/modules/redis"
)

// StartRedis spins up a Redis container for integration tests.
//
// Skips when testing.Short() is set (containers are slow and need Docker).
// Termination is registered via t.Cleanup; the returned cleanup func is also
// safe to call explicitly (idempotent after first terminate).
//
// Prefer miniredis or in-memory adapters for unit tests; use this only when
// exercising a real Redis protocol.
func StartRedis(t *testing.T) (string, func()) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping redis container in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	redisContainer, err := redis.Run(ctx,
		"redis:7-alpine",
		redis.WithSnapshotting(0, 0),
		redis.WithLogLevel(redis.LogLevelVerbose),
	)
	if err != nil {
		t.Fatalf("failed to start redis container: %v", err)
	}

	connStr, err := redisContainer.ConnectionString(ctx)
	if err != nil {
		_ = redisContainer.Terminate(context.Background())
		t.Fatalf("failed to get connection string: %v", err)
	}

	var once sync.Once
	cleanup := func() {
		once.Do(func() {
			if err := redisContainer.Terminate(context.Background()); err != nil {
				t.Logf("failed to terminate redis container: %v", err)
			}
		})
	}
	t.Cleanup(cleanup)

	return connStr, cleanup
}

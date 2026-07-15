package test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// StartPostgres spins up a Postgres container for integration tests.
//
// Skips when testing.Short() is set (containers are slow and need Docker).
// Termination is registered via t.Cleanup; the returned cleanup func is also
// safe to call explicitly (idempotent after first terminate).
//
// Prefer in-memory adapters for unit tests; use this only when exercising a
// real Postgres wire protocol.
func StartPostgres(t *testing.T) (string, func()) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping postgres container in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	pgContainer, err := postgres.Run(ctx,
		"postgres:15-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("user"),
		postgres.WithPassword("password"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(15*time.Second)),
	)
	if err != nil {
		t.Fatalf("failed to start postgres container: %v", err)
	}

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		_ = pgContainer.Terminate(context.Background())
		t.Fatalf("failed to get connection string: %v", err)
	}

	var once sync.Once
	cleanup := func() {
		once.Do(func() {
			if err := pgContainer.Terminate(context.Background()); err != nil {
				t.Logf("failed to terminate postgres container: %v", err)
			}
		})
	}
	t.Cleanup(cleanup)

	return connStr, cleanup
}

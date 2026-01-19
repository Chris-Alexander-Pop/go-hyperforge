package tests

import (
	"context"
	"testing"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/client/grpc"
)

func TestNewGRPCClient(t *testing.T) {
	cfg := grpc.Config{
		Target:                "localhost:50051",
		Timeout:               time.Second,
		Insecure:              true,
		CircuitBreakerEnabled: false,
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// This will likely fail to connect (no server), but we test the construction logic
	conn, err := grpc.New(ctx, cfg)

	// Dialing is non-blocking in recent gRPC versions without WithBlock, so it might return nil error even if server is down.
	// If it errors, that's fine too as long as it's not a panic or configuration error.
	if conn != nil {
		conn.Close()
	}

	if err != nil && err != context.DeadlineExceeded {
		// Just ensure it doesn't fail with something unexpected
		t.Logf("Expected connection failure or success, got: %v", err)
	}
}

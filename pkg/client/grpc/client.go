package grpc

import (
	"context"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Config struct {
	Target  string        `env:"CLIENT_GRPC_TARGET" env-required:"true"` // e.g. "localhost:9090" or "dns:///..."
	Timeout time.Duration `env:"CLIENT_TIMEOUT" env-default:"5s"`
}

// New creates a robust gRPC connection
func New(ctx context.Context, cfg Config) (*grpc.ClientConn, error) {
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()), // Default insecure for internal
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	}

	// Set timeout for the Dial
	ctx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()

	conn, err := grpc.DialContext(ctx, cfg.Target, opts...)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

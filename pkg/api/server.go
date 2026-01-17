// Package api provides unified API server abstractions for multiple transports.
//
// Supported transports:
//   - REST: HTTP/JSON APIs using Echo framework
//   - gRPC: High-performance RPC using Protocol Buffers
//   - GraphQL: Query language for APIs
//
// Usage:
//
//	import "github.com/chris-alexander-pop/system-design-library/pkg/api"
//
//	server, err := api.New(api.Config{Protocol: api.ProtocolREST, Port: "8080"})
//	server.Start()
package api

import (
	"context"
	"fmt"

	"github.com/chris-alexander-pop/system-design-library/pkg/api/grpc"
	"github.com/chris-alexander-pop/system-design-library/pkg/api/rest"
	"github.com/labstack/echo/v4"
)

type Protocol string

const (
	ProtocolREST    Protocol = "rest"
	ProtocolGRPC    Protocol = "grpc"
	ProtocolGraphQL Protocol = "graphql"
)

// Config for the unified API Server
type Config struct {
	Protocol Protocol `env:"API_PROTOCOL" env-default:"rest"`
	Port     string   `env:"PORT" env-default:"8080"`

	// Server specific configs could be nested or flattened.
	// For simplicity, we reuse Port. Real world might differ.
}

// Server interface for any transport
type Server interface {
	Start() error
	Shutdown(ctx context.Context) error
}

// New creates a new API server based on configuration
func New(cfg Config) (Server, error) {
	switch cfg.Protocol {
	case ProtocolREST:
		return rest.New(rest.Config{Port: cfg.Port}), nil

	case ProtocolGRPC:
		g := grpc.New(grpc.Config{Port: cfg.Port})
		return &grpcServerWrapper{g}, nil

	case ProtocolGraphQL:
		// GraphQL is typically REST (HTTP) serving a Handler.
		// We use REST server but mount GraphQL handler.
		// Specialized setup requires schemas here, which this factory doesn't know.
		// So this is a stub or we assume simple playground for now.
		r := rest.New(rest.Config{Port: cfg.Port})
		r.Echo().Any("/query", func(c echo.Context) error { return nil }) // Stub
		return r, nil

	default:
		return nil, fmt.Errorf("unknown protocol: %s", cfg.Protocol)
	}
}

// Wrapper for gRPC to match Server interface (Stop vs Shutdown)
type grpcServerWrapper struct {
	s *grpc.Server
}

func (w *grpcServerWrapper) Start() error {
	return w.s.Start()
}

func (w *grpcServerWrapper) Shutdown(ctx context.Context) error {
	// graceful stop doesn't take context in basic grpc, but we can simulate or just call it.
	// Assuming w.s.Stop() does graceful.
	w.s.Stop()
	return nil
}

package api

import (
	"context"
	"fmt"

	"github.com/99designs/gqlgen/graphql"
	apigraphql "github.com/chris-alexander-pop/go-hyperforge/pkg/api/graphql"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/api/grpc"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/api/rest"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/labstack/echo/v4"
)

type Protocol string

const (
	ProtocolREST    Protocol = "rest"
	ProtocolGRPC    Protocol = "grpc"
	ProtocolGraphQL Protocol = "graphql"
)

// Config for the unified API Server.
type Config struct {
	Protocol Protocol `env:"API_PROTOCOL" env-default:"rest"`
	Port     string   `env:"PORT" env-default:"8080"`

	// GraphQLSchema is required when Protocol is ProtocolGraphQL.
	// Typically produced by gqlgen codegen (ExecutableSchema).
	GraphQLSchema graphql.ExecutableSchema

	// GraphQLPath is the HTTP path for the GraphQL endpoint (default "/query").
	GraphQLPath string

	// GraphQLPlaygroundPath, when non-empty, mounts the GraphQL playground at that path.
	GraphQLPlaygroundPath string
}

// Server interface for any transport.
type Server interface {
	Start() error
	Shutdown(ctx context.Context) error
}

// New creates a new API server based on configuration.
func New(cfg Config) (Server, error) {
	switch cfg.Protocol {
	case ProtocolREST:
		return rest.New(rest.Config{Port: cfg.Port}), nil

	case ProtocolGRPC:
		g := grpc.New(grpc.Config{Port: cfg.Port})
		return &grpcServerWrapper{g}, nil

	case ProtocolGraphQL:
		if cfg.GraphQLSchema == nil {
			return nil, errors.InvalidArgument("GraphQLSchema is required for ProtocolGraphQL", nil)
		}
		path := cfg.GraphQLPath
		if path == "" {
			path = "/query"
		}
		r := rest.New(rest.Config{Port: cfg.Port})
		r.Echo().Any(path, echo.WrapHandler(apigraphql.NewHandler(cfg.GraphQLSchema)))
		if cfg.GraphQLPlaygroundPath != "" {
			r.Echo().GET(cfg.GraphQLPlaygroundPath, echo.WrapHandler(apigraphql.NewPlaygroundHandler(path)))
		}
		return r, nil

	default:
		return nil, errors.InvalidArgument(fmt.Sprintf("unknown protocol: %s", cfg.Protocol), nil)
	}
}

// grpcServerWrapper adapts grpc.Server to the Server interface (Stop vs Shutdown).
type grpcServerWrapper struct {
	s *grpc.Server
}

func (w *grpcServerWrapper) Start() error {
	return w.s.Start()
}

func (w *grpcServerWrapper) Shutdown(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		w.s.Stop()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

package graphql

import (
	"context"
	"net/http"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/chris-alexander-pop/system-design-library/pkg/logger"
)

// NewHandler creates a robust GraphQL HTTP handler
// schema: The generated ExecutableSchema
func NewHandler(schema graphql.ExecutableSchema) http.Handler {
	srv := handler.NewDefaultServer(schema)

	// Add OTel Tracing if package available (commented out to avoid dep hell unless requested)
	// srv.Use(otelgqlgen.Middleware())

	// Add Custom Logging Middleware/Extension
	srv.AroundOperations(func(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler {
		// Log start
		oc := graphql.GetOperationContext(ctx)
		logger.L().InfoContext(ctx, "graphql op start", "name", oc.OperationName)

		return next(ctx)
	})

	return srv
}

func NewPlaygroundHandler(endpoint string) http.Handler {
	return playground.Handler("GraphQL Playground", endpoint)
}

package graphql

import (
	"context"
	"fmt"
	"net/http"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// DefaultComplexityLimit is applied when HandlerConfig.ComplexityLimit is 0.
const DefaultComplexityLimit = 200

// DefaultDepthLimit is applied when HandlerConfig.DepthLimit is 0.
const DefaultDepthLimit = 15

// HandlerConfig tunes GraphQL request guards and observability.
type HandlerConfig struct {
	// ComplexityLimit caps query complexity (gqlgen FixedComplexityLimit).
	// Zero uses DefaultComplexityLimit. Negative disables the limit.
	ComplexityLimit int

	// DepthLimit caps selection-set nesting depth via AroundFields.
	// Zero uses DefaultDepthLimit. Negative disables the limit.
	DepthLimit int

	// DisableOTel skips OpenTelemetry spans around operations.
	DisableOTel bool
}

// DefaultHandlerConfig returns conservative production defaults.
func DefaultHandlerConfig() HandlerConfig {
	return HandlerConfig{
		ComplexityLimit: DefaultComplexityLimit,
		DepthLimit:      DefaultDepthLimit,
	}
}

// NewHandler creates a GraphQL HTTP handler with default complexity/depth limits
// and OpenTelemetry operation spans.
func NewHandler(schema graphql.ExecutableSchema) http.Handler {
	return NewHandlerWithConfig(schema, DefaultHandlerConfig())
}

// NewHandlerWithConfig creates a GraphQL HTTP handler with the given config.
func NewHandlerWithConfig(schema graphql.ExecutableSchema, cfg HandlerConfig) http.Handler {
	srv := handler.NewDefaultServer(schema)

	complexityLimit := cfg.ComplexityLimit
	if complexityLimit == 0 {
		complexityLimit = DefaultComplexityLimit
	}
	if complexityLimit > 0 {
		srv.Use(extension.FixedComplexityLimit(complexityLimit))
	}

	depthLimit := cfg.DepthLimit
	if depthLimit == 0 {
		depthLimit = DefaultDepthLimit
	}
	if depthLimit > 0 {
		limit := depthLimit
		srv.AroundFields(func(ctx context.Context, next graphql.Resolver) (any, error) {
			fc := graphql.GetFieldContext(ctx)
			if fc != nil {
				depth := fieldDepth(fc)
				if depth > limit {
					return nil, fmt.Errorf("query depth %d exceeds limit of %d", depth, limit)
				}
			}
			return next(ctx)
		})
	}

	tracer := otel.Tracer("pkg/api/graphql")
	srv.AroundOperations(func(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler {
		oc := graphql.GetOperationContext(ctx)
		opName := ""
		if oc != nil {
			opName = oc.OperationName
		}
		logger.L().InfoContext(ctx, "graphql op start", "name", opName)

		if cfg.DisableOTel {
			return next(ctx)
		}

		ctx, span := tracer.Start(ctx, "graphql.operation",
			trace.WithAttributes(attribute.String("graphql.operation.name", opName)),
		)
		handler := next(ctx)
		return func(ctx context.Context) *graphql.Response {
			defer span.End()
			resp := handler(ctx)
			if resp != nil && len(resp.Errors) > 0 {
				span.SetStatus(codes.Error, resp.Errors[0].Message)
			}
			return resp
		}
	})

	return srv
}

// fieldDepth counts nested object fields (list indices do not add depth).
func fieldDepth(fc *graphql.FieldContext) int {
	depth := 0
	for it := fc; it != nil; it = it.Parent {
		if it.Index != nil {
			continue
		}
		if it.Field.Field != nil {
			depth++
		}
	}
	return depth
}

// NewPlaygroundHandler mounts the GraphQL playground UI for the given endpoint.
func NewPlaygroundHandler(endpoint string) http.Handler {
	return playground.Handler("GraphQL Playground", endpoint)
}

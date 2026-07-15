package graphql

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/logger"
	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
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

	// EnableIntrospection enables GraphQL introspection (__schema / __type).
	// Defaults to true when unset via DefaultHandlerConfig; set false in production.
	EnableIntrospection bool
}

// DefaultHandlerConfig returns conservative production defaults.
// Introspection is enabled for local DX; disable explicitly for production.
func DefaultHandlerConfig() HandlerConfig {
	return HandlerConfig{
		ComplexityLimit:     DefaultComplexityLimit,
		DepthLimit:          DefaultDepthLimit,
		EnableIntrospection: true,
	}
}

// NewHandler creates a GraphQL HTTP handler with default complexity/depth limits
// and OpenTelemetry operation spans.
func NewHandler(schema graphql.ExecutableSchema) http.Handler {
	return NewHandlerWithConfig(schema, DefaultHandlerConfig())
}

// NewHandlerWithConfig creates a GraphQL HTTP handler with the given config.
func NewHandlerWithConfig(schema graphql.ExecutableSchema, cfg HandlerConfig) http.Handler {
	srv := handler.New(schema)

	srv.AddTransport(transport.Websocket{KeepAlivePingInterval: 10 * time.Second})
	srv.AddTransport(transport.Options{})
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})
	srv.AddTransport(transport.MultipartForm{})
	srv.SetQueryCache(lru.New[*ast.QueryDocument](1000))

	if cfg.EnableIntrospection {
		srv.Use(extension.Introspection{})
	}
	srv.Use(extension.AutomaticPersistedQuery{
		Cache: lru.New[string](100),
	})

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

// PlaygroundConfig customizes the GraphQL Playground / GraphiQL UI.
type PlaygroundConfig struct {
	// Title shown in the browser tab (default "GraphQL Playground").
	Title string
	// Endpoint is the GraphQL HTTP endpoint path or URL.
	Endpoint string
	// FetcherHeaders are sent by the playground fetcher (not shown in UI).
	FetcherHeaders map[string]string
	// UIHeaders are default headers shown in the UI headers editor.
	UIHeaders map[string]string
	// EnablePluginExplorer enables the GraphiQL plugin explorer.
	EnablePluginExplorer bool
	// StoragePrefix namespaces playground localStorage keys.
	StoragePrefix string
}

// NewPlaygroundHandler mounts the GraphQL playground UI for the given endpoint.
func NewPlaygroundHandler(endpoint string) http.Handler {
	return NewPlaygroundHandlerWithConfig(PlaygroundConfig{Endpoint: endpoint})
}

// NewPlaygroundHandlerWithConfig mounts playground with DX options.
func NewPlaygroundHandlerWithConfig(cfg PlaygroundConfig) http.Handler {
	title := cfg.Title
	if title == "" {
		title = "GraphQL Playground"
	}
	endpoint := cfg.Endpoint
	if endpoint == "" {
		endpoint = "/query"
	}

	opts := make([]playground.GraphiqlConfigOption, 0, 4)
	if len(cfg.FetcherHeaders) > 0 {
		opts = append(opts, playground.WithGraphiqlFetcherHeaders(cfg.FetcherHeaders))
	}
	if len(cfg.UIHeaders) > 0 {
		opts = append(opts, playground.WithGraphiqlUiHeaders(cfg.UIHeaders))
	}
	if cfg.EnablePluginExplorer {
		opts = append(opts, playground.WithGraphiqlEnablePluginExplorer(true))
	}
	if cfg.StoragePrefix != "" {
		opts = append(opts, playground.WithStoragePrefix(cfg.StoragePrefix))
	}
	return playground.Handler(title, endpoint, opts...)
}

// SchemaRegistry stores named SDL documents for multi-schema DX and tests.
type SchemaRegistry struct {
	mu      sync.RWMutex
	schemas map[string]string
}

// NewSchemaRegistry creates an empty registry.
func NewSchemaRegistry() *SchemaRegistry {
	return &SchemaRegistry{schemas: make(map[string]string)}
}

// Register stores SDL text under name.
func (r *SchemaRegistry) Register(name, sdl string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.schemas[name] = sdl
}

// RegisterFile reads an SDL file and registers it under name.
func (r *SchemaRegistry) RegisterFile(name, path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("graphql: read schema %q: %w", path, err)
	}
	r.Register(name, string(b))
	return nil
}

// Get returns registered SDL by name.
func (r *SchemaRegistry) Get(name string) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.schemas[name]
	return s, ok
}

// Names returns registered schema names.
func (r *SchemaRegistry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.schemas))
	for k := range r.schemas {
		out = append(out, k)
	}
	return out
}

// LoadSDL parses GraphQL SDL text into an *ast.Schema.
func LoadSDL(sdl string) (*ast.Schema, error) {
	return gqlparser.LoadSchema(&ast.Source{Name: "schema.graphql", Input: sdl})
}

// LoadSDLFile reads and parses a GraphQL SDL file.
func LoadSDLFile(path string) (*ast.Schema, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("graphql: load SDL file %q: %w", path, err)
	}
	return LoadSDL(string(b))
}

// MustLoadSDLFile is like LoadSDLFile but panics on error (tests/boot).
func MustLoadSDLFile(path string) *ast.Schema {
	s, err := LoadSDLFile(path)
	if err != nil {
		panic(err)
	}
	return s
}

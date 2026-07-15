// Package graphql wraps gqlgen to build an HTTP GraphQL handler with operation
// logging, OpenTelemetry spans, complexity limits, and selection-set depth limits.
//
// Defaults (Override via NewHandlerWithConfig):
//   - ComplexityLimit: 200 (gqlgen FixedComplexityLimit; negative disables)
//   - DepthLimit: 15 (AroundFields; negative disables)
//   - OTel: operation spans on tracer "pkg/api/graphql"
//
// Provide a gqlgen ExecutableSchema (usually from codegen). Schema loading and
// custom error formatters remain the application's responsibility.
//
// Usage:
//
//	h := graphql.NewHandler(generated.NewExecutableSchema(...))
//	http.Handle("/query", h)
//	http.Handle("/", graphql.NewPlaygroundHandler("/query"))
//
//	// Or with custom limits:
//	h = graphql.NewHandlerWithConfig(schema, graphql.HandlerConfig{
//		ComplexityLimit: 100,
//		DepthLimit:      10,
//	})
//
// Or mount via the unified factory:
//
//	api.New(api.Config{Protocol: api.ProtocolGraphQL, GraphQLSchema: schema, Port: "8080"})
package graphql

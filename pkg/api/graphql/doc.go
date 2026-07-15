// Package graphql wraps gqlgen to build an HTTP GraphQL handler with operation
// logging, OpenTelemetry spans, complexity/depth limits, and introspection control.
//
// Defaults (override via NewHandlerWithConfig):
//   - ComplexityLimit: 200 (gqlgen FixedComplexityLimit; negative disables)
//   - DepthLimit: 15 (AroundFields; negative disables)
//   - EnableIntrospection: true (disable in production)
//   - OTel: operation spans on tracer "pkg/api/graphql"
//
// DX helpers:
//   - SchemaRegistry / LoadSDL / LoadSDLFile for SDL loading
//   - NewPlaygroundHandlerWithConfig for GraphiQL title/headers/explorer options
//
// Provide a gqlgen ExecutableSchema (usually from codegen) for the HTTP handler.
//
// Usage:
//
//	h := graphql.NewHandler(generated.NewExecutableSchema(...))
//	http.Handle("/query", h)
//	http.Handle("/", graphql.NewPlaygroundHandler("/query"))
//
//	// Or with custom limits / introspection off:
//	h = graphql.NewHandlerWithConfig(schema, graphql.HandlerConfig{
//		ComplexityLimit:     100,
//		DepthLimit:          10,
//		EnableIntrospection: false,
//	})
//
// Or mount via the unified factory:
//
//	api.New(api.Config{Protocol: api.ProtocolGraphQL, GraphQLSchema: schema, Port: "8080"})
package graphql

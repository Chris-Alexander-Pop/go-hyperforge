// Package graphql wraps gqlgen to build an HTTP GraphQL handler with operation logging
// and an optional playground handler.
//
// Provide a gqlgen ExecutableSchema (usually from codegen). Schema loading, complexity
// limits, and custom error formatters are the application's responsibility via gqlgen
// options on the returned handler or schema config.
//
// Usage:
//
//	h := graphql.NewHandler(generated.NewExecutableSchema(...))
//	http.Handle("/query", h)
//	http.Handle("/", graphql.NewPlaygroundHandler("/query"))
//
// Or mount via the unified factory:
//
//	api.New(api.Config{Protocol: api.ProtocolGraphQL, GraphQLSchema: schema, Port: "8080"})
package graphql

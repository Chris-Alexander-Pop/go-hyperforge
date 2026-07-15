/*
Package api is the unified entrypoint and factory for REST, gRPC, and GraphQL transports.

Capabilities wired today: REST timeouts + HTTPStatus errors, gRPC health/recovery/GRPCStatus,
GraphQL schema injection, WebSocket hub (rooms + upgrade auth), RBAC middleware, multi-key
rate limiting, pkg/api/openapi document helpers from route metadata, and Echo↔stdlib bridges.

Not covered by this factory: production GraphQL complexity/auth plugins (configure via gqlgen).

Usage:

	server, err := api.New(api.Config{Protocol: api.ProtocolREST, Port: "8080"})

For GraphQL, supply an ExecutableSchema (typically from gqlgen codegen):

	server, err := api.New(api.Config{
		Protocol:      api.ProtocolGraphQL,
		Port:          "8080",
		GraphQLSchema: generated.NewExecutableSchema(...),
	})
*/
package api

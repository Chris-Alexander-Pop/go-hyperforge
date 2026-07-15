/*
Package api is the unified entrypoint and factory for REST, gRPC, and GraphQL transports.

Capabilities wired today: REST timeouts + HTTPStatus errors, gRPC health/recovery/GRPCStatus,
GraphQL schema injection with complexity/depth limits + OTel spans, WebSocket hub (rooms +
upgrade auth), RBAC middleware, multi-key rate limiting, pkg/api/openapi document helpers
from route metadata, and Echo↔stdlib bridges.

GraphQL defaults (complexity 200, depth 15, OTel on) live in pkg/api/graphql; override via
graphql.NewHandlerWithConfig when mounting outside this factory.

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

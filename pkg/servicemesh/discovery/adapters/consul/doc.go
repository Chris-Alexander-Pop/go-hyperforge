// Package consul implements discovery.ServiceRegistry against Consul's HTTP agent/catalog API.
//
// Supported operations use the agent and health endpoints (no official Consul SDK):
//
//	PUT  /v1/agent/service/register
//	PUT  /v1/agent/service/deregister/{id}
//	GET  /v1/health/service/{name}
//	GET  /v1/agent/service/{id}
//	GET  /v1/agent/services
//	PUT  /v1/agent/check/pass/service:{id}
//
// Watch uses Consul blocking queries (?index=&wait=). Pass Config.Address as the
// Consul HTTP base URL (e.g. http://127.0.0.1:8500). Tests use net/http/httptest.
package consul

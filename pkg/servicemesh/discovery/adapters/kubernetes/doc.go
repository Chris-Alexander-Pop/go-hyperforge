// Package kubernetes provides a discovery.ServiceRegistry backed by Kubernetes Endpoints.
//
// Register/Deregister manage Endpoints subsets for a Service name; Lookup/Watch
// read Endpoints (and optionally EndpointSlices when available). Tests use
// client-go fake clientsets. This is a thin client — not a full controller.
package kubernetes

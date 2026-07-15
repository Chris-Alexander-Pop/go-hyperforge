// Package etcd provides a discovery.ServiceRegistry backed by the etcd v3 HTTP JSON API.
//
// Service instances are stored as JSON under /{prefix}/services/{name}/{id}.
// Tests use httptest; production points Address at an etcd gRPC-gateway
// (e.g. http://127.0.0.1:2379). This is a thin KV registry, not a lease/election client.
package etcd

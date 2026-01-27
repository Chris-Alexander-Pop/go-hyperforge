// Package discovery provides a unified interface for service discovery and registration.
//
// Supported backends:
//   - Memory: In-memory registry for testing
//   - Consul: HashiCorp Consul
//   - Etcd: etcd key-value store
//   - Kubernetes: Kubernetes service discovery
//   - Eureka: Netflix Eureka
//
// Usage:
//
//	import "github.com/chris-alexander-pop/system-design-library/pkg/servicemesh/discovery/adapters/memory"
//
//	registry := memory.New()
//	err := registry.Register(ctx, discovery.Service{Name: "api", Address: "10.0.0.1", Port: 8080})
package discovery

import (
	"context"
	"time"
)

// Driver constants for discovery backends.
const (
	DriverMemory     = "memory"
	DriverConsul     = "consul"
	DriverEtcd       = "etcd"
	DriverKubernetes = "kubernetes"
	DriverEureka     = "eureka"
)

// HealthStatus represents the health of a service instance.
type HealthStatus string

const (
	HealthStatusPassing  HealthStatus = "passing"
	HealthStatusWarning  HealthStatus = "warning"
	HealthStatusCritical HealthStatus = "critical"
	HealthStatusUnknown  HealthStatus = "unknown"
)

// Config holds configuration for service discovery.
type Config struct {
	// Driver specifies the discovery backend.
	Driver string `env:"DISCOVERY_DRIVER" env-default:"memory"`

	// Consul specific
	ConsulAddress string `env:"CONSUL_ADDRESS" env-default:"localhost:8500"`
	ConsulToken   string `env:"CONSUL_TOKEN"`

	// Etcd specific
	EtcdEndpoints []string `env:"ETCD_ENDPOINTS"`
	EtcdUsername  string   `env:"ETCD_USERNAME"`
	EtcdPassword  string   `env:"ETCD_PASSWORD"`

	// Common options
	Namespace       string        `env:"DISCOVERY_NAMESPACE" env-default:"default"`
	TTL             time.Duration `env:"DISCOVERY_TTL" env-default:"30s"`
	RefreshInterval time.Duration `env:"DISCOVERY_REFRESH" env-default:"10s"`
}

// Service represents a registered service instance.
type Service struct {
	// ID is the unique instance identifier.
	ID string

	// Name is the service name.
	Name string

	// Address is the service address (IP or hostname).
	Address string

	// Port is the service port.
	Port int

	// Tags are service tags for filtering.
	Tags []string

	// Metadata is key-value metadata.
	Metadata map[string]string

	// Health is the current health status.
	Health HealthStatus

	// Namespace is the service namespace.
	Namespace string

	// Weight is for weighted load balancing.
	Weight int

	// RegisteredAt is when the service registered.
	RegisteredAt time.Time

	// LastHeartbeat is the last health check time.
	LastHeartbeat time.Time
}

// RegisterOptions configures service registration.
type RegisterOptions struct {
	// ID is the instance identifier (auto-generated if empty).
	ID string

	// Name is the service name.
	Name string

	// Address is the service address.
	Address string

	// Port is the service port.
	Port int

	// Tags are service tags.
	Tags []string

	// Metadata is key-value metadata.
	Metadata map[string]string

	// HealthCheck configures health checking.
	HealthCheck *HealthCheck

	// TTL is the registration TTL.
	TTL time.Duration

	// Weight for weighted load balancing.
	Weight int
}

// HealthCheck configures health checking.
type HealthCheck struct {
	// Type is the check type (http, tcp, grpc, ttl).
	Type string

	// Interval is the check interval.
	Interval time.Duration

	// Timeout is the check timeout.
	Timeout time.Duration

	// HTTP is the HTTP endpoint to check.
	HTTP string

	// GRPC is the gRPC service to check.
	GRPC string

	// TCP is the TCP address to check.
	TCP string

	// DeregisterCriticalServiceAfter removes unhealthy services.
	DeregisterCriticalServiceAfter time.Duration
}

// QueryOptions configures service queries.
type QueryOptions struct {
	// Tag filters by tag.
	Tag string

	// Namespace filters by namespace.
	Namespace string

	// HealthyOnly returns only healthy instances.
	HealthyOnly bool

	// Limit is the maximum results.
	Limit int
}

// ServiceRegistry defines the interface for service discovery.
type ServiceRegistry interface {
	// Register registers a service instance.
	Register(ctx context.Context, opts RegisterOptions) (*Service, error)

	// Deregister removes a service instance.
	Deregister(ctx context.Context, serviceID string) error

	// Lookup finds service instances by name.
	Lookup(ctx context.Context, serviceName string, opts QueryOptions) ([]*Service, error)

	// Get retrieves a specific service instance.
	Get(ctx context.Context, serviceID string) (*Service, error)

	// List returns all registered services.
	List(ctx context.Context, opts QueryOptions) ([]*Service, error)

	// Watch watches for service changes.
	Watch(ctx context.Context, serviceName string) (<-chan []*Service, error)

	// Heartbeat sends a health heartbeat for a service.
	Heartbeat(ctx context.Context, serviceID string) error

	// UpdateHealth updates the health status of a service.
	UpdateHealth(ctx context.Context, serviceID string, status HealthStatus) error

	// Close closes the registry connection.
	Close() error
}

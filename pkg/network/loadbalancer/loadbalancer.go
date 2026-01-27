// Package loadbalancer provides a unified interface for load balancer management.
//
// Supported backends:
//   - Memory: In-memory load balancer for testing
//   - ALB: AWS Application Load Balancer
//   - NLB: AWS Network Load Balancer
//   - GCLB: Google Cloud Load Balancing
//   - AzureLB: Azure Load Balancer
//
// Usage:
//
//	import "github.com/chris-alexander-pop/system-design-library/pkg/network/loadbalancer/adapters/memory"
//
//	lb := memory.New()
//	err := lb.AddTarget(ctx, "pool-1", loadbalancer.Target{Address: "10.0.0.1", Port: 8080})
package loadbalancer

import (
	"context"
	"time"
)

// Driver constants for load balancer backends.
const (
	DriverMemory  = "memory"
	DriverALB     = "alb"
	DriverNLB     = "nlb"
	DriverGCLB    = "gclb"
	DriverAzureLB = "azure-lb"
	DriverHAProxy = "haproxy"
	DriverNginx   = "nginx"
)

// Protocol represents the load balancer protocol.
type Protocol string

const (
	ProtocolHTTP  Protocol = "HTTP"
	ProtocolHTTPS Protocol = "HTTPS"
	ProtocolTCP   Protocol = "TCP"
	ProtocolUDP   Protocol = "UDP"
	ProtocolTLS   Protocol = "TLS"
)

// Algorithm represents the load balancing algorithm.
type Algorithm string

const (
	AlgorithmRoundRobin         Algorithm = "round-robin"
	AlgorithmLeastConnections   Algorithm = "least-connections"
	AlgorithmWeightedRoundRobin Algorithm = "weighted-round-robin"
	AlgorithmIPHash             Algorithm = "ip-hash"
	AlgorithmRandom             Algorithm = "random"
)

// HealthCheckType represents the health check protocol.
type HealthCheckType string

const (
	HealthCheckHTTP  HealthCheckType = "HTTP"
	HealthCheckHTTPS HealthCheckType = "HTTPS"
	HealthCheckTCP   HealthCheckType = "TCP"
	HealthCheckGRPC  HealthCheckType = "GRPC"
)

// TargetStatus represents the health status of a target.
type TargetStatus string

const (
	TargetStatusHealthy   TargetStatus = "healthy"
	TargetStatusUnhealthy TargetStatus = "unhealthy"
	TargetStatusDraining  TargetStatus = "draining"
	TargetStatusUnused    TargetStatus = "unused"
)

// Config holds configuration for load balancer management.
type Config struct {
	// Driver specifies the load balancer backend.
	Driver string `env:"LB_DRIVER" env-default:"memory"`

	// AWS specific
	AWSAccessKeyID     string `env:"LB_AWS_ACCESS_KEY"`
	AWSSecretAccessKey string `env:"LB_AWS_SECRET_KEY"`
	AWSRegion          string `env:"LB_AWS_REGION" env-default:"us-east-1"`

	// GCP specific
	GCPProjectID string `env:"LB_GCP_PROJECT"`

	// Azure specific
	AzureSubscriptionID string `env:"LB_AZURE_SUBSCRIPTION"`
	AzureResourceGroup  string `env:"LB_AZURE_RESOURCE_GROUP"`

	// Common options
	DefaultHealthCheckInterval time.Duration `env:"LB_HEALTH_INTERVAL" env-default:"30s"`
	DefaultHealthCheckTimeout  time.Duration `env:"LB_HEALTH_TIMEOUT" env-default:"5s"`
	DefaultHealthyThreshold    int           `env:"LB_HEALTHY_THRESHOLD" env-default:"2"`
	DefaultUnhealthyThreshold  int           `env:"LB_UNHEALTHY_THRESHOLD" env-default:"2"`
}

// LoadBalancer represents a load balancer instance.
type LoadBalancer struct {
	// ID is the unique identifier.
	ID string

	// Name is the load balancer name.
	Name string

	// DNSName is the DNS name for the load balancer.
	DNSName string

	// Scheme is "internet-facing" or "internal".
	Scheme string

	// Type is the load balancer type (application, network, etc.).
	Type string

	// State is the current state.
	State string

	// Listeners are the attached listeners.
	Listeners []*Listener

	// Tags are key-value metadata.
	Tags map[string]string

	// CreatedAt is when the load balancer was created.
	CreatedAt time.Time
}

// Listener represents a load balancer listener.
type Listener struct {
	// ID is the unique identifier.
	ID string

	// LoadBalancerID is the parent load balancer.
	LoadBalancerID string

	// Protocol is the listener protocol.
	Protocol Protocol

	// Port is the listening port.
	Port int

	// TargetPoolID is the default target pool.
	TargetPoolID string

	// SSLCertificateARN is the SSL certificate (for HTTPS/TLS).
	SSLCertificateARN string

	// Rules are routing rules for the listener.
	Rules []*Rule

	// CreatedAt is when the listener was created.
	CreatedAt time.Time
}

// Rule represents a routing rule.
type Rule struct {
	// ID is the unique identifier.
	ID string

	// Priority is the rule priority (lower = higher priority).
	Priority int

	// Conditions are the matching conditions.
	Conditions []RuleCondition

	// TargetPoolID is the target pool for matched requests.
	TargetPoolID string
}

// RuleCondition represents a condition for routing.
type RuleCondition struct {
	// Field is the field to match (path-pattern, host-header, etc.).
	Field string

	// Values are the values to match.
	Values []string
}

// TargetPool represents a pool of targets (target group).
type TargetPool struct {
	// ID is the unique identifier.
	ID string

	// Name is the pool name.
	Name string

	// Protocol is the target protocol.
	Protocol Protocol

	// Port is the target port.
	Port int

	// Algorithm is the load balancing algorithm.
	Algorithm Algorithm

	// HealthCheck configures health checking.
	HealthCheck *HealthCheck

	// Targets are the pool members.
	Targets []*Target

	// Tags are key-value metadata.
	Tags map[string]string

	// CreatedAt is when the pool was created.
	CreatedAt time.Time
}

// HealthCheck configures health checking.
type HealthCheck struct {
	// Type is the health check type.
	Type HealthCheckType

	// Path is the HTTP path to check.
	Path string

	// Port is the port to check.
	Port int

	// IntervalSeconds is the check interval.
	IntervalSeconds int

	// TimeoutSeconds is the check timeout.
	TimeoutSeconds int

	// HealthyThreshold is successful checks to mark healthy.
	HealthyThreshold int

	// UnhealthyThreshold is failed checks to mark unhealthy.
	UnhealthyThreshold int

	// ExpectedCodes are acceptable HTTP status codes.
	ExpectedCodes string
}

// Target represents a backend target.
type Target struct {
	// ID is the unique identifier.
	ID string

	// Address is the target IP or hostname.
	Address string

	// Port is the target port.
	Port int

	// Weight is the target weight for weighted routing.
	Weight int

	// Status is the health status.
	Status TargetStatus

	// Reason explains unhealthy status.
	Reason string

	// RegisteredAt is when the target was registered.
	RegisteredAt time.Time
}

// CreateLoadBalancerOptions configures load balancer creation.
type CreateLoadBalancerOptions struct {
	Name     string
	Scheme   string
	Type     string
	Tags     map[string]string
	Subnets  []string
	Security []string
}

// CreateListenerOptions configures listener creation.
type CreateListenerOptions struct {
	LoadBalancerID    string
	Protocol          Protocol
	Port              int
	TargetPoolID      string
	SSLCertificateARN string
}

// CreateTargetPoolOptions configures target pool creation.
type CreateTargetPoolOptions struct {
	Name        string
	Protocol    Protocol
	Port        int
	Algorithm   Algorithm
	HealthCheck *HealthCheck
	Tags        map[string]string
}

// LoadBalancerManager defines the interface for load balancer operations.
type LoadBalancerManager interface {
	// CreateLoadBalancer creates a new load balancer.
	CreateLoadBalancer(ctx context.Context, opts CreateLoadBalancerOptions) (*LoadBalancer, error)

	// GetLoadBalancer retrieves a load balancer by ID.
	GetLoadBalancer(ctx context.Context, id string) (*LoadBalancer, error)

	// ListLoadBalancers returns all load balancers.
	ListLoadBalancers(ctx context.Context) ([]*LoadBalancer, error)

	// DeleteLoadBalancer deletes a load balancer.
	DeleteLoadBalancer(ctx context.Context, id string) error

	// CreateListener creates a listener on a load balancer.
	CreateListener(ctx context.Context, opts CreateListenerOptions) (*Listener, error)

	// DeleteListener removes a listener.
	DeleteListener(ctx context.Context, loadBalancerID, listenerID string) error

	// CreateTargetPool creates a target pool (target group).
	CreateTargetPool(ctx context.Context, opts CreateTargetPoolOptions) (*TargetPool, error)

	// GetTargetPool retrieves a target pool by ID.
	GetTargetPool(ctx context.Context, id string) (*TargetPool, error)

	// DeleteTargetPool deletes a target pool.
	DeleteTargetPool(ctx context.Context, id string) error

	// AddTarget adds a target to a pool.
	AddTarget(ctx context.Context, poolID string, target Target) error

	// RemoveTarget removes a target from a pool.
	RemoveTarget(ctx context.Context, poolID, targetID string) error

	// GetTargetHealth returns the health of all targets in a pool.
	GetTargetHealth(ctx context.Context, poolID string) ([]*Target, error)

	// AddRule adds a routing rule to a listener.
	AddRule(ctx context.Context, listenerID string, rule Rule) (*Rule, error)

	// RemoveRule removes a routing rule.
	RemoveRule(ctx context.Context, listenerID, ruleID string) error
}

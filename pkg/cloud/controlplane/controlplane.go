package controlplane

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/cloud"
)

// ControlPlane serves as the central brain of the private cloud.
// It tracks the state of all hosts and instances.
type ControlPlane interface {
	// RegisterHost adds a new host to the cluster.
	RegisterHost(ctx context.Context, host cloud.Host) error

	// DeregisterHost removes a host from the cluster.
	DeregisterHost(ctx context.Context, hostID string) error

	// UpdateHostStatus updates the status of a known host.
	UpdateHostStatus(ctx context.Context, hostID string, status cloud.HostStatus) error

	// GetHost retrieves a host by ID.
	GetHost(ctx context.Context, hostID string) (*cloud.Host, error)

	// ListHosts returns a list of all registered hosts.
	ListHosts(ctx context.Context) ([]cloud.Host, error)

	// CreateInstance registers a new instance, optionally bound to a host.
	// When HostID is set on the request, capacity is reserved on that host.
	CreateInstance(ctx context.Context, req CreateInstanceRequest) (*Instance, error)

	// BindInstance assigns an unbound instance to a host (or rebinds).
	BindInstance(ctx context.Context, instanceID, hostID string) error

	// UnbindInstance detaches an instance from its host without deleting it.
	UnbindInstance(ctx context.Context, instanceID string) error

	// GetInstance retrieves an instance by ID.
	GetInstance(ctx context.Context, instanceID string) (*Instance, error)

	// ListInstances returns instances, optionally filtered by host.
	ListInstances(ctx context.Context, opts ListInstancesOptions) ([]Instance, error)

	// DeleteInstance removes an instance and releases host capacity.
	DeleteInstance(ctx context.Context, instanceID string) error

	// UpdateInstanceStatus updates instance lifecycle status.
	UpdateInstanceStatus(ctx context.Context, instanceID string, status cloud.InstanceStatus) error
}

// Instance is a control-plane tracked VM/workload placement record.
type Instance struct {
	ID        string               `json:"id"`
	Name      string               `json:"name"`
	HostID    string               `json:"host_id,omitempty"`
	Status    cloud.InstanceStatus `json:"status"`
	Resources cloud.Resources      `json:"resources"`
	Image     string               `json:"image,omitempty"`
	Tags      map[string]string    `json:"tags,omitempty"`
	CreatedAt time.Time            `json:"created_at"`
}

// CreateInstanceRequest configures instance creation / host binding.
type CreateInstanceRequest struct {
	Name      string
	HostID    string // optional; when set, instance is bound immediately
	Resources cloud.Resources
	Image     string
	Tags      map[string]string
}

// ListInstancesOptions filters instance listing.
type ListInstancesOptions struct {
	HostID string
	Status cloud.InstanceStatus
}

// Config holds configuration for the Control Plane.
type Config struct {
	// Driver specifies the storage backend for state: "memory", "etcd", "postgres".
	// memory and etcd (HTTP JSON API) are implemented; postgres is reserved.
	Driver string `env:"CONTROLPLANE_DRIVER" env-default:"memory"`
}

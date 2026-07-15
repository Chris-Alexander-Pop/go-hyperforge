package provisioning

import (
	"context"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/cloud"
)

// Provisioner defines the interface for managing bare metal host lifecycles.
// This typically interfaces with technologies like PXE, IPMI, or Redfish.
type Provisioner interface {
	// ProvisionHost provisions an OS onto a bare metal host.
	ProvisionHost(ctx context.Context, hostID string, imageURL string) error

	// DeprovisionHost wipes a host and returns it to the pool.
	DeprovisionHost(ctx context.Context, hostID string) error

	// GetHostStatus returns the current provisioning/power status of a host.
	GetHostStatus(ctx context.Context, hostID string) (cloud.HostStatus, error)

	// PowerCycle reboots a host (cold restart).
	PowerCycle(ctx context.Context, hostID string) error
}

// Config holds configuration for the Provisioner.
type Config struct {
	// Driver specifies the provisioning backend: "memory", "ipmi", "pxe".
	Driver string `env:"PROVISIONING_DRIVER" env-default:"memory"`
}

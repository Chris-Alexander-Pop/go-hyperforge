package hypervisor

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/cloud"
)

// Hypervisor defines the interface for managing Virtual Machines on a host.
// Implementations can be local (e.g. Libvirt) or remote.
type Hypervisor interface {
	// CreateVM creates a new virtual machine with the given specification.
	// Returns the ID of the created VM.
	CreateVM(ctx context.Context, spec VMSpec) (string, error)

	// StartVM powers on an existing virtual machine works.
	StartVM(ctx context.Context, vmID string) error

	// StopVM gracefully shuts down a virtual machine.
	StopVM(ctx context.Context, vmID string) error

	// DeleteVM removes a virtual machine and its resources.
	DeleteVM(ctx context.Context, vmID string) error

	// GetVMStatus returns the current status of a virtual machine.
	GetVMStatus(ctx context.Context, vmID string) (cloud.InstanceStatus, error)

	// ListVMs returns a list of all VMs managed by this hypervisor.
	ListVMs(ctx context.Context) ([]VM, error)
}

// VMSpec defines the requirements for a new Virtual Machine.
type VMSpec struct {
	Name         string             `json:"name"`
	InstanceType cloud.InstanceType `json:"instance_type"`
	Image        string             `json:"image"` // OS Image / ISO path
	MemoryMB     int                `json:"memory_mb"`
	VCPUs        int                `json:"vcpus"`
	Tags         map[string]string  `json:"tags,omitempty"`
}

// VM represents a running Virtual Machine instance.
type VM struct {
	ID        string               `json:"id"`
	Name      string               `json:"name"`
	Status    cloud.InstanceStatus `json:"status"`
	Spec      VMSpec               `json:"spec"`
	CreatedAt time.Time            `json:"created_at"`
	IPAddress string               `json:"ip_address,omitempty"`
}

// Config holds configuration for the Hypervisor.
type Config struct {
	// Driver specifies the hypervisor backend: "memory" or "libvirt".
	Driver string `env:"HYPERVISOR_DRIVER" env-default:"memory"`

	// URI is the connection string for the hypervisor (e.g., "qemu:///system").
	URI string `env:"HYPERVISOR_URI" env-default:"qemu:///system"`
}

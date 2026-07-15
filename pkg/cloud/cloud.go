package cloud

// Common constants and types for the Cloud domain (private-cloud IaaS).
//
// These types are shared across hypervisor, provisioning, scheduler, and
// controlplane. They are intentionally separate from pkg/compute/vm Instance
// types, which mirror public-cloud provider APIs.

// HostStatus represents the state of a physical host or hypervisor.
type HostStatus string

const (
	HostStatusUnknown     HostStatus = "unknown"
	HostStatusReady       HostStatus = "ready"
	HostStatusMaintenance HostStatus = "maintenance"
	HostStatusOffline     HostStatus = "offline"
	HostStatusBusy        HostStatus = "busy"
)

// InstanceStatus represents the state of a Virtual Machine.
type InstanceStatus string

const (
	InstanceStatusUnknown      InstanceStatus = "unknown"
	InstanceStatusPending      InstanceStatus = "pending"
	InstanceStatusRunning      InstanceStatus = "running"
	InstanceStatusStopped      InstanceStatus = "stopped"
	InstanceStatusTerminated   InstanceStatus = "terminated"
	InstanceStatusProvisioning InstanceStatus = "provisioning"
	InstanceStatusError        InstanceStatus = "error"
)

// InstanceType defines the size/capability class of a VM (e.g. t3.micro equivalent).
type InstanceType string

const (
	InstanceTypeSmall  InstanceType = "small"  // 1 vCPU, 2GB RAM
	InstanceTypeMedium InstanceType = "medium" // 2 vCPU, 4GB RAM
	InstanceTypeLarge  InstanceType = "large"  // 4 vCPU, 8GB RAM
	InstanceTypeXLarge InstanceType = "xlarge" // 8 vCPU, 16GB RAM
)

// Resources represents compute resources.
type Resources struct {
	VCPUs    int `json:"vcpus"`
	MemoryMB int `json:"memory_mb"`
	DiskGB   int `json:"disk_gb"`
	GPUs     int `json:"gpus,omitempty"`
}

// Host represents a physical machine or hypervisor node.
type Host struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Status    HostStatus        `json:"status"`
	Capacity  Resources         `json:"capacity"`
	Available Resources         `json:"available"`
	Zone      string            `json:"zone"`
	Tags      map[string]string `json:"tags,omitempty"`
}

// Driver definitions for cloud components.
const (
	DriverMemory      = "memory"
	DriverLibvirt     = "libvirt"     // remote JSON/HTTP gateway (pure Go)
	DriverFirecracker = "firecracker" // HTTP over unix socket
	DriverIPMI        = "ipmi"
	DriverRedfish     = "redfish"
	DriverPXE         = "pxe" // reserved
)

package controller

import (
	"context"
	"time"
)

// VolumeController defines the interface for managing the lifecycle of storage volumes.
// It abstracts operations like provisioning, attaching, and snapshotting block devices.
type VolumeController interface {
	// CreateVolume provisions a new storage volume.
	CreateVolume(ctx context.Context, spec VolumeSpec) (string, error)

	// DeleteVolume removes a storage volume.
	DeleteVolume(ctx context.Context, volumeID string) error

	// AttachVolume attaches a volume to a specific host or VM.
	AttachVolume(ctx context.Context, volumeID string, nodeID string) error

	// DetachVolume detaches a volume from its current host.
	DetachVolume(ctx context.Context, volumeID string) error

	// ResizeVolume expands an existing volume to a new size in GB.
	ResizeVolume(ctx context.Context, volumeID string, newSizeGB int) error

	// GetVolume retrieves details about a specific volume.
	GetVolume(ctx context.Context, volumeID string) (*Volume, error)
}

// VolumeSpec defines the parameters for creating a new volume.
type VolumeSpec struct {
	Name       string            `json:"name"`
	SizeGB     int               `json:"size_gb"`
	Type       string            `json:"type"` // e.g., "ssd", "hdd", "nvme"
	Zone       string            `json:"zone"`
	SnapshotID string            `json:"snapshot_id,omitempty"` // Create from snapshot
	Tags       map[string]string `json:"tags,omitempty"`
}

// Volume represents a provisioned storage volume.
type Volume struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	SizeGB     int               `json:"size_gb"`
	Type       string            `json:"type"`
	Zone       string            `json:"zone"`
	Status     VolumeStatus      `json:"status"`
	AttachedTo string            `json:"attached_to,omitempty"` // Node ID
	CreatedAt  time.Time         `json:"created_at"`
	Tags       map[string]string `json:"tags,omitempty"`
}

// VolumeStatus represents the state of a volume.
type VolumeStatus string

const (
	VolumeStatusAvailable VolumeStatus = "available"
	VolumeStatusCreating  VolumeStatus = "creating"
	VolumeStatusAttached  VolumeStatus = "attached"
	VolumeStatusDetaching VolumeStatus = "detaching"
	VolumeStatusDeleting  VolumeStatus = "deleting"
	VolumeStatusError     VolumeStatus = "error"
)

// Driver constants for volume controller backends.
const (
	DriverMemory = "memory"
	DriverLVM    = "lvm"
	DriverLocal  = "local"
	DriverCeph   = "ceph"
	DriverCSI    = "csi"
)

// Config holds configuration for the storage controller.
type Config struct {
	// Driver specifies the storage backend: "memory", "lvm", "local", "ceph", "csi".
	Driver string `env:"STORAGE_DRIVER" env-default:"memory"`

	// Root is used by lvm/local file-backed controllers.
	Root string `env:"STORAGE_CONTROLLER_ROOT" env-default:"./lvmstore"`

	// VolumeGroup is cosmetic for the lvm adapter.
	VolumeGroup string `env:"STORAGE_LVM_VG" env-default:"hyperforge"`
}

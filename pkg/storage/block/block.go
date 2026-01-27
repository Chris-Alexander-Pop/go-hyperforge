// Package block provides a unified interface for block storage (volumes).
//
// Supported backends:
//   - Memory: In-memory volume store for testing
//   - EBS: AWS Elastic Block Store (planned)
//   - Azure Disk: Azure Managed Disks (planned)
//   - Persistent Disk: GCP Persistent Disks (planned)
//
// Usage:
//
//	import "github.com/chris-alexander-pop/system-design-library/pkg/storage/block/adapters/memory"
//
//	store := memory.New()
//	vol, err := store.CreateVolume(ctx, block.CreateVolumeOptions{Name: "my-vol", SizeGB: 100})
package block

import (
	"context"
	"time"
)

// Driver constants for block storage backends.
const (
	DriverMemory          = "memory"
	DriverEBS             = "ebs"
	DriverAzureDisk       = "azure-disk"
	DriverGCPPersistent   = "gcp-persistent"
	DriverCeph            = "ceph"
	DriverOpenStackCinder = "cinder"
)

// VolumeState represents the lifecycle state of a volume.
type VolumeState string

const (
	VolumeStateCreating  VolumeState = "creating"
	VolumeStateAvailable VolumeState = "available"
	VolumeStateInUse     VolumeState = "in-use"
	VolumeStateDeleting  VolumeState = "deleting"
	VolumeStateError     VolumeState = "error"
)

// VolumeType represents the performance tier of a volume.
type VolumeType string

const (
	VolumeTypeStandard VolumeType = "standard"
	VolumeTypeSSD      VolumeType = "ssd"
	VolumeTypeIOPS     VolumeType = "provisioned-iops"
)

// Config holds configuration for block storage.
type Config struct {
	// Driver specifies the block storage backend.
	Driver string `env:"BLOCK_DRIVER" env-default:"memory"`

	// Region is the cloud region for the volumes.
	Region string `env:"BLOCK_REGION" env-default:"us-east-1"`

	// AvailabilityZone is the AZ for the volumes.
	AvailabilityZone string `env:"BLOCK_AZ"`

	// EBS specific
	AWSAccessKeyID     string `env:"BLOCK_AWS_ACCESS_KEY"`
	AWSSecretAccessKey string `env:"BLOCK_AWS_SECRET_KEY"`

	// Azure specific
	AzureSubscriptionID string `env:"BLOCK_AZURE_SUBSCRIPTION_ID"`
	AzureResourceGroup  string `env:"BLOCK_AZURE_RESOURCE_GROUP"`

	// GCP specific
	GCPProjectID string `env:"BLOCK_GCP_PROJECT_ID"`

	// Common options
	DefaultVolumeType VolumeType    `env:"BLOCK_DEFAULT_TYPE" env-default:"standard"`
	Timeout           time.Duration `env:"BLOCK_TIMEOUT" env-default:"5m"`
}

// Volume represents a block storage volume.
type Volume struct {
	// ID is the unique identifier for the volume.
	ID string

	// Name is the human-readable name.
	Name string

	// SizeGB is the volume size in gigabytes.
	SizeGB int64

	// State is the current lifecycle state.
	State VolumeState

	// VolumeType is the performance tier.
	VolumeType VolumeType

	// AvailabilityZone is where the volume is located.
	AvailabilityZone string

	// Attachments lists instances the volume is attached to.
	Attachments []Attachment

	// Encrypted indicates if the volume is encrypted.
	Encrypted bool

	// IOPS is the provisioned IOPS (for provisioned-iops type).
	IOPS int64

	// Throughput is the provisioned throughput in MB/s.
	Throughput int64

	// CreatedAt is when the volume was created.
	CreatedAt time.Time

	// Tags are key-value metadata.
	Tags map[string]string
}

// Attachment represents a volume attachment to an instance.
type Attachment struct {
	// InstanceID is the attached instance.
	InstanceID string

	// Device is the device name (e.g., /dev/sdf).
	Device string

	// AttachedAt is when the attachment occurred.
	AttachedAt time.Time
}

// CreateVolumeOptions configures volume creation.
type CreateVolumeOptions struct {
	// Name is the human-readable name for the volume.
	Name string

	// SizeGB is the requested size in gigabytes.
	SizeGB int64

	// VolumeType is the performance tier.
	VolumeType VolumeType

	// AvailabilityZone is where to create the volume.
	AvailabilityZone string

	// Encrypted enables encryption at rest.
	Encrypted bool

	// EncryptionKeyID is the KMS key for encryption (optional).
	EncryptionKeyID string

	// IOPS is the provisioned IOPS (for provisioned-iops type).
	IOPS int64

	// Throughput is the provisioned throughput in MB/s.
	Throughput int64

	// SnapshotID creates the volume from a snapshot.
	SnapshotID string

	// Tags are key-value metadata.
	Tags map[string]string
}

// ResizeVolumeOptions configures volume resizing.
type ResizeVolumeOptions struct {
	// NewSizeGB is the new size in gigabytes (must be >= current size).
	NewSizeGB int64

	// NewVolumeType optionally changes the volume type.
	NewVolumeType VolumeType

	// NewIOPS optionally changes provisioned IOPS.
	NewIOPS int64

	// NewThroughput optionally changes provisioned throughput.
	NewThroughput int64
}

// AttachVolumeOptions configures volume attachment.
type AttachVolumeOptions struct {
	// VolumeID is the volume to attach.
	VolumeID string

	// InstanceID is the instance to attach to.
	InstanceID string

	// Device is the device name (e.g., /dev/sdf).
	Device string
}

// ListOptions configures volume listing.
type ListOptions struct {
	// Limit is the maximum number of volumes to return.
	Limit int

	// NextToken is the pagination token.
	NextToken string

	// Filters are key-value filters.
	Filters map[string]string
}

// ListResult contains the list result with pagination.
type ListResult struct {
	// Volumes is the list of volumes.
	Volumes []*Volume

	// NextToken is the pagination token for the next page.
	NextToken string
}

// Snapshot represents a point-in-time snapshot of a volume.
type Snapshot struct {
	// ID is the unique identifier for the snapshot.
	ID string

	// VolumeID is the source volume.
	VolumeID string

	// SizeGB is the snapshot size in gigabytes.
	SizeGB int64

	// State is the current lifecycle state.
	State string

	// CreatedAt is when the snapshot was created.
	CreatedAt time.Time

	// Description is the snapshot description.
	Description string

	// Tags are key-value metadata.
	Tags map[string]string
}

// CreateSnapshotOptions configures snapshot creation.
type CreateSnapshotOptions struct {
	// VolumeID is the volume to snapshot.
	VolumeID string

	// Description is the snapshot description.
	Description string

	// Tags are key-value metadata.
	Tags map[string]string
}

// VolumeStore defines the interface for block storage operations.
type VolumeStore interface {
	// CreateVolume creates a new block storage volume.
	CreateVolume(ctx context.Context, opts CreateVolumeOptions) (*Volume, error)

	// GetVolume retrieves a volume by ID.
	// Returns errors.NotFound if the volume does not exist.
	GetVolume(ctx context.Context, volumeID string) (*Volume, error)

	// ListVolumes returns volumes matching the options.
	ListVolumes(ctx context.Context, opts ListOptions) (*ListResult, error)

	// DeleteVolume deletes a volume.
	// The volume must be detached first.
	DeleteVolume(ctx context.Context, volumeID string) error

	// ResizeVolume modifies a volume's size or type.
	ResizeVolume(ctx context.Context, volumeID string, opts ResizeVolumeOptions) (*Volume, error)

	// AttachVolume attaches a volume to an instance.
	AttachVolume(ctx context.Context, opts AttachVolumeOptions) error

	// DetachVolume detaches a volume from an instance.
	DetachVolume(ctx context.Context, volumeID, instanceID string) error

	// CreateSnapshot creates a point-in-time snapshot.
	CreateSnapshot(ctx context.Context, opts CreateSnapshotOptions) (*Snapshot, error)

	// GetSnapshot retrieves a snapshot by ID.
	GetSnapshot(ctx context.Context, snapshotID string) (*Snapshot, error)

	// DeleteSnapshot deletes a snapshot.
	DeleteSnapshot(ctx context.Context, snapshotID string) error
}

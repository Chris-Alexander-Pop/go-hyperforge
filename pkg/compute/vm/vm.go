// Package vm provides a unified interface for virtual machine management.
//
// Supported backends:
//   - Memory: In-memory VM manager for testing
//   - EC2: AWS EC2 instances
//   - GCE: Google Compute Engine
//   - AzureVM: Azure Virtual Machines
//
// Usage:
//
//	import "github.com/chris-alexander-pop/system-design-library/pkg/compute/vm/adapters/memory"
//
//	manager := memory.New()
//	instance, err := manager.Create(ctx, vm.CreateOptions{Name: "web-1", Image: "ubuntu-22.04"})
package vm

import (
	"context"
	"time"
)

// Driver constants for VM backends.
const (
	DriverMemory  = "memory"
	DriverEC2     = "ec2"
	DriverGCE     = "gce"
	DriverAzureVM = "azure-vm"
)

// InstanceState represents the state of a VM instance.
type InstanceState string

const (
	InstanceStatePending     InstanceState = "pending"
	InstanceStateRunning     InstanceState = "running"
	InstanceStateStopping    InstanceState = "stopping"
	InstanceStateStopped     InstanceState = "stopped"
	InstanceStateTerminating InstanceState = "terminating"
	InstanceStateTerminated  InstanceState = "terminated"
)

// Config holds configuration for VM management.
type Config struct {
	// Driver specifies the VM backend.
	Driver string `env:"VM_DRIVER" env-default:"memory"`

	// AWS specific
	AWSAccessKeyID     string `env:"VM_AWS_ACCESS_KEY"`
	AWSSecretAccessKey string `env:"VM_AWS_SECRET_KEY"`
	AWSRegion          string `env:"VM_AWS_REGION" env-default:"us-east-1"`

	// GCP specific
	GCPProjectID string `env:"VM_GCP_PROJECT"`
	GCPZone      string `env:"VM_GCP_ZONE" env-default:"us-central1-a"`

	// Azure specific
	AzureSubscriptionID string `env:"VM_AZURE_SUBSCRIPTION"`
	AzureResourceGroup  string `env:"VM_AZURE_RESOURCE_GROUP"`

	// Common options
	DefaultInstanceType string        `env:"VM_DEFAULT_TYPE" env-default:"t3.medium"`
	Timeout             time.Duration `env:"VM_TIMEOUT" env-default:"5m"`
}

// Instance represents a virtual machine instance.
type Instance struct {
	// ID is the unique identifier.
	ID string

	// Name is the instance name.
	Name string

	// State is the current state.
	State InstanceState

	// InstanceType is the machine type (e.g., t3.medium).
	InstanceType string

	// ImageID is the image/AMI used.
	ImageID string

	// PublicIP is the public IP address.
	PublicIP string

	// PrivateIP is the private IP address.
	PrivateIP string

	// Zone is the availability zone.
	Zone string

	// VPCSubnetID is the network subnet.
	VPCSubnetID string

	// SecurityGroups are the security group IDs.
	SecurityGroups []string

	// Tags are key-value metadata.
	Tags map[string]string

	// LaunchTime is when the instance was launched.
	LaunchTime time.Time
}

// CreateOptions configures instance creation.
type CreateOptions struct {
	// Name is the instance name.
	Name string

	// InstanceType is the machine type.
	InstanceType string

	// ImageID is the image to use.
	ImageID string

	// KeyName is the SSH key pair name.
	KeyName string

	// SubnetID is the VPC subnet.
	SubnetID string

	// SecurityGroupIDs are security groups to attach.
	SecurityGroupIDs []string

	// Zone is the target availability zone.
	Zone string

	// Tags are key-value metadata.
	Tags map[string]string

	// UserData is initialization script.
	UserData string
}

// ListOptions configures instance listing.
type ListOptions struct {
	// State filters by instance state.
	State InstanceState

	// Tags filters by tag key-value pairs.
	Tags map[string]string

	// Limit is the maximum instances to return.
	Limit int

	// PageToken is for pagination.
	PageToken string
}

// ListResult contains the list result.
type ListResult struct {
	// Instances is the list of instances.
	Instances []*Instance

	// NextPageToken is for pagination.
	NextPageToken string
}

// VMManager defines the interface for VM management.
type VMManager interface {
	// Create creates a new VM instance.
	Create(ctx context.Context, opts CreateOptions) (*Instance, error)

	// Get retrieves an instance by ID.
	Get(ctx context.Context, instanceID string) (*Instance, error)

	// List returns instances matching the options.
	List(ctx context.Context, opts ListOptions) (*ListResult, error)

	// Start starts a stopped instance.
	Start(ctx context.Context, instanceID string) error

	// Stop stops a running instance.
	Stop(ctx context.Context, instanceID string) error

	// Reboot reboots an instance.
	Reboot(ctx context.Context, instanceID string) error

	// Terminate terminates (deletes) an instance.
	Terminate(ctx context.Context, instanceID string) error

	// UpdateTags updates instance tags.
	UpdateTags(ctx context.Context, instanceID string, tags map[string]string) error

	// GetConsoleOutput retrieves console output.
	GetConsoleOutput(ctx context.Context, instanceID string) (string, error)
}

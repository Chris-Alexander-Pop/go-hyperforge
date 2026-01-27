// Package container provides a unified interface for container orchestration.
//
// Supported backends:
//   - Memory: In-memory container runtime for testing
//   - Docker: Docker Engine
//   - ECS: AWS Elastic Container Service
//   - GKE: Google Kubernetes Engine
//   - AKS: Azure Kubernetes Service
//
// Usage:
//
//	import "github.com/chris-alexander-pop/system-design-library/pkg/compute/container/adapters/memory"
//
//	runtime := memory.New()
//	container, err := runtime.Create(ctx, container.CreateOptions{Image: "nginx:latest"})
package container

import (
	"context"
	"io"
	"time"
)

// Driver constants for container backends.
const (
	DriverMemory = "memory"
	DriverDocker = "docker"
	DriverECS    = "ecs"
	DriverGKE    = "gke"
	DriverAKS    = "aks"
)

// ContainerState represents the state of a container.
type ContainerState string

const (
	ContainerStateCreated    ContainerState = "created"
	ContainerStateRunning    ContainerState = "running"
	ContainerStatePaused     ContainerState = "paused"
	ContainerStateRestarting ContainerState = "restarting"
	ContainerStateExited     ContainerState = "exited"
	ContainerStateDead       ContainerState = "dead"
)

// RestartPolicy defines container restart behavior.
type RestartPolicy string

const (
	RestartPolicyNo            RestartPolicy = "no"
	RestartPolicyAlways        RestartPolicy = "always"
	RestartPolicyOnFailure     RestartPolicy = "on-failure"
	RestartPolicyUnlessStopped RestartPolicy = "unless-stopped"
)

// Config holds configuration for container runtime.
type Config struct {
	// Driver specifies the container backend.
	Driver string `env:"CONTAINER_DRIVER" env-default:"memory"`

	// DockerHost is the Docker daemon socket.
	DockerHost string `env:"DOCKER_HOST" env-default:"unix:///var/run/docker.sock"`

	// AWS ECS specific
	AWSAccessKeyID     string `env:"CONTAINER_AWS_ACCESS_KEY"`
	AWSSecretAccessKey string `env:"CONTAINER_AWS_SECRET_KEY"`
	AWSRegion          string `env:"CONTAINER_AWS_REGION" env-default:"us-east-1"`
	ECSCluster         string `env:"CONTAINER_ECS_CLUSTER"`

	// Common options
	DefaultMemory int64         `env:"CONTAINER_DEFAULT_MEMORY" env-default:"512"`
	DefaultCPU    float64       `env:"CONTAINER_DEFAULT_CPU" env-default:"0.5"`
	Timeout       time.Duration `env:"CONTAINER_TIMEOUT" env-default:"2m"`
}

// Container represents a running container.
type Container struct {
	// ID is the unique identifier.
	ID string

	// Name is the container name.
	Name string

	// Image is the container image.
	Image string

	// State is the current state.
	State ContainerState

	// ExitCode is set when container exits.
	ExitCode int

	// NetworkMode is the network configuration.
	NetworkMode string

	// IPAddress is the container IP.
	IPAddress string

	// Ports are mapped ports.
	Ports []PortMapping

	// Env are environment variables.
	Env map[string]string

	// Labels are key-value metadata.
	Labels map[string]string

	// CPU is the CPU limit.
	CPU float64

	// Memory is the memory limit in MB.
	Memory int64

	// CreatedAt is when the container was created.
	CreatedAt time.Time

	// StartedAt is when the container started.
	StartedAt time.Time

	// FinishedAt is when the container stopped.
	FinishedAt time.Time
}

// PortMapping represents a port mapping.
type PortMapping struct {
	// ContainerPort is the internal port.
	ContainerPort int

	// HostPort is the external port.
	HostPort int

	// Protocol is tcp or udp.
	Protocol string
}

// VolumeMount represents a volume mount.
type VolumeMount struct {
	// Source is the host path or volume name.
	Source string

	// Target is the container path.
	Target string

	// ReadOnly makes the mount read-only.
	ReadOnly bool
}

// CreateOptions configures container creation.
type CreateOptions struct {
	// Name is the container name.
	Name string

	// Image is the container image.
	Image string

	// Command overrides the default command.
	Command []string

	// Entrypoint overrides the default entrypoint.
	Entrypoint []string

	// Env are environment variables.
	Env map[string]string

	// Labels are key-value metadata.
	Labels map[string]string

	// Ports are port mappings.
	Ports []PortMapping

	// Volumes are volume mounts.
	Volumes []VolumeMount

	// NetworkMode is the network configuration.
	NetworkMode string

	// CPU is the CPU limit.
	CPU float64

	// Memory is the memory limit in MB.
	Memory int64

	// RestartPolicy defines restart behavior.
	RestartPolicy RestartPolicy

	// WorkDir is the working directory.
	WorkDir string

	// User is the user to run as.
	User string
}

// ListOptions configures container listing.
type ListOptions struct {
	// All includes stopped containers.
	All bool

	// Labels filters by labels.
	Labels map[string]string

	// Limit is the maximum containers to return.
	Limit int
}

// ExecOptions configures command execution.
type ExecOptions struct {
	// Command is the command to run.
	Command []string

	// Env are additional environment variables.
	Env map[string]string

	// WorkDir is the working directory.
	WorkDir string

	// User is the user to run as.
	User string

	// Tty allocates a TTY.
	Tty bool
}

// ExecResult contains the execution result.
type ExecResult struct {
	// ExitCode is the command exit code.
	ExitCode int

	// Stdout is the standard output.
	Stdout string

	// Stderr is the standard error.
	Stderr string
}

// ContainerRuntime defines the interface for container operations.
type ContainerRuntime interface {
	// Create creates a new container.
	Create(ctx context.Context, opts CreateOptions) (*Container, error)

	// Get retrieves a container by ID.
	Get(ctx context.Context, containerID string) (*Container, error)

	// List returns containers matching the options.
	List(ctx context.Context, opts ListOptions) ([]*Container, error)

	// Start starts a created container.
	Start(ctx context.Context, containerID string) error

	// Stop stops a running container.
	Stop(ctx context.Context, containerID string, timeout time.Duration) error

	// Kill sends a signal to a container.
	Kill(ctx context.Context, containerID string, signal string) error

	// Remove removes a container.
	Remove(ctx context.Context, containerID string, force bool) error

	// Logs retrieves container logs.
	Logs(ctx context.Context, containerID string, follow bool) (io.ReadCloser, error)

	// Exec executes a command in a running container.
	Exec(ctx context.Context, containerID string, opts ExecOptions) (*ExecResult, error)

	// Wait waits for a container to exit.
	Wait(ctx context.Context, containerID string) (int, error)

	// Stats returns container resource statistics.
	Stats(ctx context.Context, containerID string) (*ContainerStats, error)
}

// ContainerStats contains resource usage statistics.
type ContainerStats struct {
	// CPUPercent is CPU usage percentage.
	CPUPercent float64

	// MemoryUsage is memory usage in bytes.
	MemoryUsage int64

	// MemoryLimit is memory limit in bytes.
	MemoryLimit int64

	// NetworkRx is bytes received.
	NetworkRx int64

	// NetworkTx is bytes sent.
	NetworkTx int64

	// Timestamp is when stats were collected.
	Timestamp time.Time
}

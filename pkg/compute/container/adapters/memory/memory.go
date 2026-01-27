package memory

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/compute/container"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"github.com/google/uuid"
)

// Runtime implements an in-memory container runtime for testing.
type Runtime struct {
	mu         sync.RWMutex
	containers map[string]*container.Container
	logs       map[string]*bytes.Buffer
	config     container.Config
}

// New creates a new in-memory container runtime.
func New() *Runtime {
	return &Runtime{
		containers: make(map[string]*container.Container),
		logs:       make(map[string]*bytes.Buffer),
		config:     container.Config{DefaultMemory: 512, DefaultCPU: 0.5},
	}
}

func (r *Runtime) Create(ctx context.Context, opts container.CreateOptions) (*container.Container, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check for name conflict
	if opts.Name != "" {
		for _, c := range r.containers {
			if c.Name == opts.Name {
				return nil, errors.Conflict("container name already in use", nil)
			}
		}
	}

	memory := opts.Memory
	if memory <= 0 {
		memory = r.config.DefaultMemory
	}

	cpu := opts.CPU
	if cpu <= 0 {
		cpu = r.config.DefaultCPU
	}

	ctr := &container.Container{
		ID:          uuid.NewString()[:12],
		Name:        opts.Name,
		Image:       opts.Image,
		State:       container.ContainerStateCreated,
		NetworkMode: opts.NetworkMode,
		IPAddress:   fmt.Sprintf("172.17.0.%d", len(r.containers)+2),
		Ports:       opts.Ports,
		Env:         opts.Env,
		Labels:      opts.Labels,
		CPU:         cpu,
		Memory:      memory,
		CreatedAt:   time.Now(),
	}

	if ctr.Name == "" {
		ctr.Name = "container-" + ctr.ID[:8]
	}

	r.containers[ctr.ID] = ctr
	r.logs[ctr.ID] = &bytes.Buffer{}
	r.logs[ctr.ID].WriteString(fmt.Sprintf("[%s] Container %s created\n", time.Now().Format(time.RFC3339), ctr.ID))

	return ctr, nil
}

func (r *Runtime) Get(ctx context.Context, containerID string) (*container.Container, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ctr, ok := r.containers[containerID]
	if !ok {
		return nil, errors.NotFound("container not found", nil)
	}

	return ctr, nil
}

func (r *Runtime) List(ctx context.Context, opts container.ListOptions) ([]*container.Container, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*container.Container

	for _, ctr := range r.containers {
		// Filter out stopped if not All
		if !opts.All && ctr.State == container.ContainerStateExited {
			continue
		}

		// Filter by labels
		if len(opts.Labels) > 0 {
			match := true
			for k, v := range opts.Labels {
				if ctr.Labels[k] != v {
					match = false
					break
				}
			}
			if !match {
				continue
			}
		}

		result = append(result, ctr)
	}

	// Apply limit
	if opts.Limit > 0 && len(result) > opts.Limit {
		result = result[:opts.Limit]
	}

	return result, nil
}

func (r *Runtime) Start(ctx context.Context, containerID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	ctr, ok := r.containers[containerID]
	if !ok {
		return errors.NotFound("container not found", nil)
	}

	if ctr.State == container.ContainerStateRunning {
		return errors.Conflict("container is already running", nil)
	}

	ctr.State = container.ContainerStateRunning
	ctr.StartedAt = time.Now()
	r.logs[containerID].WriteString(fmt.Sprintf("[%s] Container started\n", time.Now().Format(time.RFC3339)))

	return nil
}

func (r *Runtime) Stop(ctx context.Context, containerID string, timeout time.Duration) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	ctr, ok := r.containers[containerID]
	if !ok {
		return errors.NotFound("container not found", nil)
	}

	if ctr.State != container.ContainerStateRunning {
		return errors.Conflict("container is not running", nil)
	}

	ctr.State = container.ContainerStateExited
	ctr.FinishedAt = time.Now()
	ctr.ExitCode = 0
	r.logs[containerID].WriteString(fmt.Sprintf("[%s] Container stopped\n", time.Now().Format(time.RFC3339)))

	return nil
}

func (r *Runtime) Kill(ctx context.Context, containerID string, signal string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	ctr, ok := r.containers[containerID]
	if !ok {
		return errors.NotFound("container not found", nil)
	}

	if ctr.State != container.ContainerStateRunning {
		return errors.Conflict("container is not running", nil)
	}

	ctr.State = container.ContainerStateExited
	ctr.FinishedAt = time.Now()
	ctr.ExitCode = 137 // Killed
	r.logs[containerID].WriteString(fmt.Sprintf("[%s] Container killed with %s\n", time.Now().Format(time.RFC3339), signal))

	return nil
}

func (r *Runtime) Remove(ctx context.Context, containerID string, force bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	ctr, ok := r.containers[containerID]
	if !ok {
		return errors.NotFound("container not found", nil)
	}

	if !force && ctr.State == container.ContainerStateRunning {
		return errors.Conflict("container is running, use force to remove", nil)
	}

	delete(r.containers, containerID)
	delete(r.logs, containerID)

	return nil
}

func (r *Runtime) Logs(ctx context.Context, containerID string, follow bool) (io.ReadCloser, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if _, ok := r.containers[containerID]; !ok {
		return nil, errors.NotFound("container not found", nil)
	}

	logs, ok := r.logs[containerID]
	if !ok {
		return nil, errors.NotFound("logs not found", nil)
	}

	return io.NopCloser(strings.NewReader(logs.String())), nil
}

func (r *Runtime) Exec(ctx context.Context, containerID string, opts container.ExecOptions) (*container.ExecResult, error) {
	r.mu.RLock()
	ctr, ok := r.containers[containerID]
	r.mu.RUnlock()

	if !ok {
		return nil, errors.NotFound("container not found", nil)
	}

	if ctr.State != container.ContainerStateRunning {
		return nil, errors.Conflict("container is not running", nil)
	}

	// Simulate command execution
	stdout := fmt.Sprintf("Executed: %s\n", strings.Join(opts.Command, " "))

	return &container.ExecResult{
		ExitCode: 0,
		Stdout:   stdout,
		Stderr:   "",
	}, nil
}

func (r *Runtime) Wait(ctx context.Context, containerID string) (int, error) {
	r.mu.RLock()
	ctr, ok := r.containers[containerID]
	r.mu.RUnlock()

	if !ok {
		return -1, errors.NotFound("container not found", nil)
	}

	// Simulate waiting
	if ctr.State == container.ContainerStateRunning {
		<-ctx.Done()
		return -1, ctx.Err()
	}

	return ctr.ExitCode, nil
}

func (r *Runtime) Stats(ctx context.Context, containerID string) (*container.ContainerStats, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ctr, ok := r.containers[containerID]
	if !ok {
		return nil, errors.NotFound("container not found", nil)
	}

	if ctr.State != container.ContainerStateRunning {
		return nil, errors.Conflict("container is not running", nil)
	}

	return &container.ContainerStats{
		CPUPercent:  15.5,
		MemoryUsage: ctr.Memory * 1024 * 1024 / 2, // 50% usage
		MemoryLimit: ctr.Memory * 1024 * 1024,
		NetworkRx:   1024 * 100,
		NetworkTx:   512 * 100,
		Timestamp:   time.Now(),
	}, nil
}

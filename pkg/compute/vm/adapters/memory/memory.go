package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/compute/vm"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"github.com/google/uuid"
)

// Manager implements an in-memory VM manager for testing.
type Manager struct {
	mu        sync.RWMutex
	instances map[string]*vm.Instance
	config    vm.Config
}

// New creates a new in-memory VM manager.
func New() *Manager {
	return &Manager{
		instances: make(map[string]*vm.Instance),
		config:    vm.Config{DefaultInstanceType: "t3.medium"},
	}
}

func (m *Manager) Create(ctx context.Context, opts vm.CreateOptions) (*vm.Instance, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	instanceType := opts.InstanceType
	if instanceType == "" {
		instanceType = m.config.DefaultInstanceType
	}

	instance := &vm.Instance{
		ID:             uuid.NewString(),
		Name:           opts.Name,
		State:          vm.InstanceStateRunning,
		InstanceType:   instanceType,
		ImageID:        opts.ImageID,
		PublicIP:       fmt.Sprintf("54.%d.%d.%d", rnd(1, 255), rnd(1, 255), rnd(1, 255)),
		PrivateIP:      fmt.Sprintf("10.0.%d.%d", rnd(0, 255), rnd(1, 255)),
		Zone:           opts.Zone,
		VPCSubnetID:    opts.SubnetID,
		SecurityGroups: opts.SecurityGroupIDs,
		Tags:           opts.Tags,
		LaunchTime:     time.Now(),
	}

	if instance.Zone == "" {
		instance.Zone = "us-east-1a"
	}

	m.instances[instance.ID] = instance
	return instance, nil
}

func rnd(min, max int) int {
	return min + int(time.Now().UnixNano()%int64(max-min+1))
}

func (m *Manager) Get(ctx context.Context, instanceID string) (*vm.Instance, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	instance, ok := m.instances[instanceID]
	if !ok {
		return nil, errors.NotFound("instance not found", nil)
	}

	return instance, nil
}

func (m *Manager) List(ctx context.Context, opts vm.ListOptions) (*vm.ListResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := &vm.ListResult{
		Instances: make([]*vm.Instance, 0),
	}

	for _, instance := range m.instances {
		// Filter by state
		if opts.State != "" && instance.State != opts.State {
			continue
		}

		// Filter by tags
		if len(opts.Tags) > 0 {
			match := true
			for k, v := range opts.Tags {
				if instance.Tags[k] != v {
					match = false
					break
				}
			}
			if !match {
				continue
			}
		}

		result.Instances = append(result.Instances, instance)
	}

	// Apply limit
	if opts.Limit > 0 && len(result.Instances) > opts.Limit {
		result.Instances = result.Instances[:opts.Limit]
		result.NextPageToken = "more"
	}

	return result, nil
}

func (m *Manager) Start(ctx context.Context, instanceID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	instance, ok := m.instances[instanceID]
	if !ok {
		return errors.NotFound("instance not found", nil)
	}

	if instance.State != vm.InstanceStateStopped {
		return errors.Conflict("instance must be stopped to start", nil)
	}

	instance.State = vm.InstanceStateRunning
	return nil
}

func (m *Manager) Stop(ctx context.Context, instanceID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	instance, ok := m.instances[instanceID]
	if !ok {
		return errors.NotFound("instance not found", nil)
	}

	if instance.State != vm.InstanceStateRunning {
		return errors.Conflict("instance must be running to stop", nil)
	}

	instance.State = vm.InstanceStateStopped
	return nil
}

func (m *Manager) Reboot(ctx context.Context, instanceID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	instance, ok := m.instances[instanceID]
	if !ok {
		return errors.NotFound("instance not found", nil)
	}

	if instance.State != vm.InstanceStateRunning {
		return errors.Conflict("instance must be running to reboot", nil)
	}

	// Simulated reboot - state stays running
	return nil
}

func (m *Manager) Terminate(ctx context.Context, instanceID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	instance, ok := m.instances[instanceID]
	if !ok {
		return errors.NotFound("instance not found", nil)
	}

	if instance.State == vm.InstanceStateTerminated {
		return errors.Conflict("instance already terminated", nil)
	}

	instance.State = vm.InstanceStateTerminated
	return nil
}

func (m *Manager) UpdateTags(ctx context.Context, instanceID string, tags map[string]string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	instance, ok := m.instances[instanceID]
	if !ok {
		return errors.NotFound("instance not found", nil)
	}

	if instance.Tags == nil {
		instance.Tags = make(map[string]string)
	}

	for k, v := range tags {
		instance.Tags[k] = v
	}

	return nil
}

func (m *Manager) GetConsoleOutput(ctx context.Context, instanceID string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	_, ok := m.instances[instanceID]
	if !ok {
		return "", errors.NotFound("instance not found", nil)
	}

	return fmt.Sprintf("[%s] Instance %s booted successfully\n", time.Now().Format(time.RFC3339), instanceID), nil
}

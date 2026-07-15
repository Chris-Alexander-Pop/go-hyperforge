// Package azurevm provides a thin Azure Virtual Machines adapter scaffold for vm.VMManager.
//
// Full ARM Compute SDK wiring is deferred; lifecycle methods that need the SDK
// return Unimplemented. Interface conformance is guaranteed for composition.
package azurevm

import (
	"context"

	"github.com/chris-alexander-pop/system-design-library/pkg/compute/vm"
	pkgerrors "github.com/chris-alexander-pop/system-design-library/pkg/errors"
)

// Config holds Azure VM adapter configuration.
type Config struct {
	SubscriptionID string `env:"VM_AZURE_SUBSCRIPTION"`
	ResourceGroup  string `env:"VM_AZURE_RESOURCE_GROUP"`
	Location       string `env:"VM_AZURE_LOCATION" env-default:"eastus"`
}

// Manager is a scaffold Azure VM manager.
type Manager struct {
	config Config
}

// New creates an Azure VM manager scaffold.
func New(cfg Config) (*Manager, error) {
	if cfg.SubscriptionID == "" {
		return nil, pkgerrors.InvalidArgument("subscription_id is required", nil)
	}
	if cfg.ResourceGroup == "" {
		return nil, pkgerrors.InvalidArgument("resource_group is required", nil)
	}
	return &Manager{config: cfg}, nil
}

func unimplemented(op string) error {
	return pkgerrors.Unimplemented("azurevm."+op+" requires Azure ARM Compute SDK wiring", nil)
}

func (m *Manager) Create(ctx context.Context, opts vm.CreateOptions) (*vm.Instance, error) {
	return nil, unimplemented("Create")
}
func (m *Manager) Get(ctx context.Context, instanceID string) (*vm.Instance, error) {
	return nil, unimplemented("Get")
}
func (m *Manager) List(ctx context.Context, opts vm.ListOptions) (*vm.ListResult, error) {
	return nil, unimplemented("List")
}
func (m *Manager) Start(ctx context.Context, instanceID string) error {
	return unimplemented("Start")
}
func (m *Manager) Stop(ctx context.Context, instanceID string) error {
	return unimplemented("Stop")
}
func (m *Manager) Reboot(ctx context.Context, instanceID string) error {
	return unimplemented("Reboot")
}
func (m *Manager) Terminate(ctx context.Context, instanceID string) error {
	return unimplemented("Terminate")
}
func (m *Manager) UpdateTags(ctx context.Context, instanceID string, tags map[string]string) error {
	return unimplemented("UpdateTags")
}
func (m *Manager) GetConsoleOutput(ctx context.Context, instanceID string) (string, error) {
	return "", unimplemented("GetConsoleOutput")
}

var _ vm.VMManager = (*Manager)(nil)

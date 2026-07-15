package vm

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/events"
	"github.com/google/uuid"
)

// Ensure EventedVMManager implements VMManager at compile time.
var _ VMManager = (*EventedVMManager)(nil)

const (
	// TopicVM is the pkg/events topic for VM domain events.
	TopicVM = "compute.vm"

	// EventTypeCreated is emitted after a successful Create.
	EventTypeCreated = "vm.created"

	// EventTypeStarted is emitted after a successful Start.
	EventTypeStarted = "vm.started"

	// EventTypeStopped is emitted after a successful Stop.
	EventTypeStopped = "vm.stopped"

	// EventTypeTerminated is emitted after a successful Terminate.
	EventTypeTerminated = "vm.terminated"
)

// InstanceEventPayload is the typed payload for VM lifecycle events.
type InstanceEventPayload struct {
	InstanceID string    `json:"instance_id"`
	Name       string    `json:"name,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
}

// EventedVMManager decorates a VMManager to publish domain events via pkg/events.
// Publish is best-effort: failures are ignored so VM ops are not rolled back.
type EventedVMManager struct {
	next VMManager
	bus  events.Bus
}

// NewEventedVMManager wraps next so Create/Start/Stop/Terminate fan out to bus.
// If bus is nil, publishing is skipped.
func NewEventedVMManager(next VMManager, bus events.Bus) *EventedVMManager {
	return &EventedVMManager{next: next, bus: bus}
}

func (m *EventedVMManager) publish(ctx context.Context, eventType, instanceID, name string) {
	if m.bus == nil {
		return
	}
	_ = m.bus.Publish(ctx, TopicVM, events.Event{
		ID:        uuid.NewString(),
		Type:      eventType,
		Source:    "pkg/compute/vm",
		Timestamp: time.Now().UTC(),
		Payload: InstanceEventPayload{
			InstanceID: instanceID,
			Name:       name,
			Timestamp:  time.Now().UTC(),
		},
	})
}

// Create delegates then publishes vm.created (best-effort).
func (m *EventedVMManager) Create(ctx context.Context, opts CreateOptions) (*Instance, error) {
	inst, err := m.next.Create(ctx, opts)
	if err != nil {
		return nil, err
	}
	m.publish(ctx, EventTypeCreated, inst.ID, inst.Name)
	return inst, nil
}

// Get delegates to the underlying manager.
func (m *EventedVMManager) Get(ctx context.Context, instanceID string) (*Instance, error) {
	return m.next.Get(ctx, instanceID)
}

// List delegates to the underlying manager.
func (m *EventedVMManager) List(ctx context.Context, opts ListOptions) (*ListResult, error) {
	return m.next.List(ctx, opts)
}

// Start delegates then publishes vm.started (best-effort).
func (m *EventedVMManager) Start(ctx context.Context, instanceID string) error {
	if err := m.next.Start(ctx, instanceID); err != nil {
		return err
	}
	m.publish(ctx, EventTypeStarted, instanceID, "")
	return nil
}

// Stop delegates then publishes vm.stopped (best-effort).
func (m *EventedVMManager) Stop(ctx context.Context, instanceID string) error {
	if err := m.next.Stop(ctx, instanceID); err != nil {
		return err
	}
	m.publish(ctx, EventTypeStopped, instanceID, "")
	return nil
}

// Reboot delegates to the underlying manager.
func (m *EventedVMManager) Reboot(ctx context.Context, instanceID string) error {
	return m.next.Reboot(ctx, instanceID)
}

// Terminate delegates then publishes vm.terminated (best-effort).
func (m *EventedVMManager) Terminate(ctx context.Context, instanceID string) error {
	if err := m.next.Terminate(ctx, instanceID); err != nil {
		return err
	}
	m.publish(ctx, EventTypeTerminated, instanceID, "")
	return nil
}

// UpdateTags delegates to the underlying manager.
func (m *EventedVMManager) UpdateTags(ctx context.Context, instanceID string, tags map[string]string) error {
	return m.next.UpdateTags(ctx, instanceID, tags)
}

// GetConsoleOutput delegates to the underlying manager.
func (m *EventedVMManager) GetConsoleOutput(ctx context.Context, instanceID string) (string, error) {
	return m.next.GetConsoleOutput(ctx, instanceID)
}

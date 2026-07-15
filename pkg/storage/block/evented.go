package block

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/events"
	"github.com/google/uuid"
)

// Ensure EventedVolumeStore implements VolumeStore at compile time.
var _ VolumeStore = (*EventedVolumeStore)(nil)

const (
	// TopicBlock is the pkg/events topic for block storage domain events.
	TopicBlock = "storage.block"

	// EventTypeVolumeCreated is emitted after CreateVolume.
	EventTypeVolumeCreated = "block.volume.created"

	// EventTypeVolumeDeleted is emitted after DeleteVolume.
	EventTypeVolumeDeleted = "block.volume.deleted"

	// EventTypeVolumeAttached is emitted after AttachVolume.
	EventTypeVolumeAttached = "block.volume.attached"

	// EventTypeVolumeDetached is emitted after DetachVolume.
	EventTypeVolumeDetached = "block.volume.detached"
)

// VolumeEventPayload is the typed payload for volume lifecycle events.
type VolumeEventPayload struct {
	VolumeID   string    `json:"volume_id"`
	InstanceID string    `json:"instance_id,omitempty"`
	Name       string    `json:"name,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
}

// EventedVolumeStore decorates a VolumeStore to publish domain events via pkg/events.
// Publish is best-effort: failures are ignored so storage ops are not rolled back.
type EventedVolumeStore struct {
	next VolumeStore
	bus  events.Bus
}

// NewEventedVolumeStore wraps next so create/delete/attach/detach fan out to bus.
// If bus is nil, publishing is skipped.
func NewEventedVolumeStore(next VolumeStore, bus events.Bus) *EventedVolumeStore {
	return &EventedVolumeStore{next: next, bus: bus}
}

func (s *EventedVolumeStore) publish(ctx context.Context, eventType, volumeID, instanceID, name string) {
	if s.bus == nil {
		return
	}
	_ = s.bus.Publish(ctx, TopicBlock, events.Event{
		ID:        uuid.NewString(),
		Type:      eventType,
		Source:    "pkg/storage/block",
		Timestamp: time.Now().UTC(),
		Payload: VolumeEventPayload{
			VolumeID:   volumeID,
			InstanceID: instanceID,
			Name:       name,
			Timestamp:  time.Now().UTC(),
		},
	})
}

// CreateVolume delegates then publishes block.volume.created (best-effort).
func (s *EventedVolumeStore) CreateVolume(ctx context.Context, opts CreateVolumeOptions) (*Volume, error) {
	vol, err := s.next.CreateVolume(ctx, opts)
	if err != nil {
		return nil, err
	}
	s.publish(ctx, EventTypeVolumeCreated, vol.ID, "", vol.Name)
	return vol, nil
}

// GetVolume delegates to the underlying store.
func (s *EventedVolumeStore) GetVolume(ctx context.Context, volumeID string) (*Volume, error) {
	return s.next.GetVolume(ctx, volumeID)
}

// ListVolumes delegates to the underlying store.
func (s *EventedVolumeStore) ListVolumes(ctx context.Context, opts ListOptions) (*ListResult, error) {
	return s.next.ListVolumes(ctx, opts)
}

// DeleteVolume delegates then publishes block.volume.deleted (best-effort).
func (s *EventedVolumeStore) DeleteVolume(ctx context.Context, volumeID string) error {
	if err := s.next.DeleteVolume(ctx, volumeID); err != nil {
		return err
	}
	s.publish(ctx, EventTypeVolumeDeleted, volumeID, "", "")
	return nil
}

// ResizeVolume delegates to the underlying store.
func (s *EventedVolumeStore) ResizeVolume(ctx context.Context, volumeID string, opts ResizeVolumeOptions) (*Volume, error) {
	return s.next.ResizeVolume(ctx, volumeID, opts)
}

// AttachVolume delegates then publishes block.volume.attached (best-effort).
func (s *EventedVolumeStore) AttachVolume(ctx context.Context, opts AttachVolumeOptions) error {
	if err := s.next.AttachVolume(ctx, opts); err != nil {
		return err
	}
	s.publish(ctx, EventTypeVolumeAttached, opts.VolumeID, opts.InstanceID, "")
	return nil
}

// DetachVolume delegates then publishes block.volume.detached (best-effort).
func (s *EventedVolumeStore) DetachVolume(ctx context.Context, volumeID, instanceID string) error {
	if err := s.next.DetachVolume(ctx, volumeID, instanceID); err != nil {
		return err
	}
	s.publish(ctx, EventTypeVolumeDetached, volumeID, instanceID, "")
	return nil
}

// CreateSnapshot delegates to the underlying store.
func (s *EventedVolumeStore) CreateSnapshot(ctx context.Context, opts CreateSnapshotOptions) (*Snapshot, error) {
	return s.next.CreateSnapshot(ctx, opts)
}

// GetSnapshot delegates to the underlying store.
func (s *EventedVolumeStore) GetSnapshot(ctx context.Context, snapshotID string) (*Snapshot, error) {
	return s.next.GetSnapshot(ctx, snapshotID)
}

// DeleteSnapshot delegates to the underlying store.
func (s *EventedVolumeStore) DeleteSnapshot(ctx context.Context, snapshotID string) error {
	return s.next.DeleteSnapshot(ctx, snapshotID)
}

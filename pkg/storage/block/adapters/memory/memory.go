package memory

import (
	"context"
	"sync"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"github.com/chris-alexander-pop/system-design-library/pkg/storage/block"
	"github.com/google/uuid"
)

// Store implements an in-memory volume store for testing.
type Store struct {
	mu        sync.RWMutex
	volumes   map[string]*block.Volume
	snapshots map[string]*block.Snapshot
}

// New creates a new in-memory volume store.
func New() *Store {
	return &Store{
		volumes:   make(map[string]*block.Volume),
		snapshots: make(map[string]*block.Snapshot),
	}
}

// NewWithConfig creates a new in-memory volume store with config (config is ignored for memory).
func NewWithConfig(_ block.Config) *Store {
	return New()
}

func (s *Store) CreateVolume(ctx context.Context, opts block.CreateVolumeOptions) (*block.Volume, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if opts.SizeGB <= 0 {
		return nil, errors.InvalidArgument("size must be > 0", nil)
	}

	volType := opts.VolumeType
	if volType == "" {
		volType = block.VolumeTypeStandard
	}

	vol := &block.Volume{
		ID:               uuid.NewString(),
		Name:             opts.Name,
		SizeGB:           opts.SizeGB,
		State:            block.VolumeStateAvailable,
		VolumeType:       volType,
		AvailabilityZone: opts.AvailabilityZone,
		Encrypted:        opts.Encrypted,
		IOPS:             opts.IOPS,
		Throughput:       opts.Throughput,
		CreatedAt:        time.Now(),
		Tags:             opts.Tags,
		Attachments:      []block.Attachment{},
	}

	if vol.Tags == nil {
		vol.Tags = make(map[string]string)
	}

	s.volumes[vol.ID] = vol
	return vol, nil
}

func (s *Store) GetVolume(ctx context.Context, volumeID string) (*block.Volume, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	vol, ok := s.volumes[volumeID]
	if !ok {
		return nil, errors.NotFound("volume not found", nil)
	}

	return vol, nil
}

func (s *Store) ListVolumes(ctx context.Context, opts block.ListOptions) (*block.ListResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := &block.ListResult{
		Volumes: make([]*block.Volume, 0, len(s.volumes)),
	}

	for _, vol := range s.volumes {
		// Apply filters
		if len(opts.Filters) > 0 {
			match := true
			for k, v := range opts.Filters {
				switch k {
				case "name":
					if vol.Name != v {
						match = false
					}
				case "state":
					if string(vol.State) != v {
						match = false
					}
				}
			}
			if !match {
				continue
			}
		}
		result.Volumes = append(result.Volumes, vol)
	}

	// Apply limit
	if opts.Limit > 0 && len(result.Volumes) > opts.Limit {
		result.Volumes = result.Volumes[:opts.Limit]
		result.NextToken = "more"
	}

	return result, nil
}

func (s *Store) DeleteVolume(ctx context.Context, volumeID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	vol, ok := s.volumes[volumeID]
	if !ok {
		return errors.NotFound("volume not found", nil)
	}

	if len(vol.Attachments) > 0 {
		return errors.Conflict("volume is attached to an instance", nil)
	}

	delete(s.volumes, volumeID)
	return nil
}

func (s *Store) ResizeVolume(ctx context.Context, volumeID string, opts block.ResizeVolumeOptions) (*block.Volume, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	vol, ok := s.volumes[volumeID]
	if !ok {
		return nil, errors.NotFound("volume not found", nil)
	}

	if opts.NewSizeGB > 0 {
		if opts.NewSizeGB < vol.SizeGB {
			return nil, errors.InvalidArgument("new size must be >= current size", nil)
		}
		vol.SizeGB = opts.NewSizeGB
	}

	if opts.NewVolumeType != "" {
		vol.VolumeType = opts.NewVolumeType
	}

	if opts.NewIOPS > 0 {
		vol.IOPS = opts.NewIOPS
	}

	if opts.NewThroughput > 0 {
		vol.Throughput = opts.NewThroughput
	}

	return vol, nil
}

func (s *Store) AttachVolume(ctx context.Context, opts block.AttachVolumeOptions) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	vol, ok := s.volumes[opts.VolumeID]
	if !ok {
		return errors.NotFound("volume not found", nil)
	}

	// Check if already attached to this instance
	for _, att := range vol.Attachments {
		if att.InstanceID == opts.InstanceID {
			return errors.Conflict("volume already attached to instance", nil)
		}
	}

	vol.Attachments = append(vol.Attachments, block.Attachment{
		InstanceID: opts.InstanceID,
		Device:     opts.Device,
		AttachedAt: time.Now(),
	})
	vol.State = block.VolumeStateInUse

	return nil
}

func (s *Store) DetachVolume(ctx context.Context, volumeID, instanceID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	vol, ok := s.volumes[volumeID]
	if !ok {
		return errors.NotFound("volume not found", nil)
	}

	found := false
	newAttachments := make([]block.Attachment, 0, len(vol.Attachments))
	for _, att := range vol.Attachments {
		if att.InstanceID == instanceID {
			found = true
			continue
		}
		newAttachments = append(newAttachments, att)
	}

	if !found {
		return errors.InvalidArgument("volume is not attached to instance", nil)
	}

	vol.Attachments = newAttachments
	if len(vol.Attachments) == 0 {
		vol.State = block.VolumeStateAvailable
	}

	return nil
}

func (s *Store) CreateSnapshot(ctx context.Context, opts block.CreateSnapshotOptions) (*block.Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	vol, ok := s.volumes[opts.VolumeID]
	if !ok {
		return nil, errors.NotFound("volume not found", nil)
	}

	snap := &block.Snapshot{
		ID:          uuid.NewString(),
		VolumeID:    opts.VolumeID,
		SizeGB:      vol.SizeGB,
		State:       "completed",
		CreatedAt:   time.Now(),
		Description: opts.Description,
		Tags:        opts.Tags,
	}

	if snap.Tags == nil {
		snap.Tags = make(map[string]string)
	}

	s.snapshots[snap.ID] = snap
	return snap, nil
}

func (s *Store) GetSnapshot(ctx context.Context, snapshotID string) (*block.Snapshot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	snap, ok := s.snapshots[snapshotID]
	if !ok {
		return nil, errors.NotFound("snapshot not found", nil)
	}

	return snap, nil
}

func (s *Store) DeleteSnapshot(ctx context.Context, snapshotID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.snapshots[snapshotID]; !ok {
		return errors.NotFound("snapshot not found", nil)
	}

	delete(s.snapshots, snapshotID)
	return nil
}

// Package local provides a file-backed block.VolumeStore for local/dev use.
// Volume and snapshot metadata are persisted as JSON under a root directory;
// this is not a real cloud block device driver.
package local

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/concurrency"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/storage/block"
	"github.com/google/uuid"
)

// Store implements block.VolumeStore with JSON files under root.
type Store struct {
	root string
	mu   *concurrency.SmartRWMutex
}

var _ block.VolumeStore = (*Store)(nil)

// New creates a local volume store rooted at rootDir.
func New(rootDir string) (*Store, error) {
	if rootDir == "" {
		return nil, errors.InvalidArgument("block local root is required", nil)
	}
	if err := os.MkdirAll(filepath.Join(rootDir, "volumes"), 0o755); err != nil {
		return nil, errors.Wrap(err, "failed to create volumes dir")
	}
	if err := os.MkdirAll(filepath.Join(rootDir, "snapshots"), 0o755); err != nil {
		return nil, errors.Wrap(err, "failed to create snapshots dir")
	}
	abs, err := filepath.Abs(rootDir)
	if err != nil {
		return nil, errors.Wrap(err, "failed to resolve block store root")
	}
	return &Store{
		root: abs,
		mu:   concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "block-local"}),
	}, nil
}

// NewWithConfig uses cfg.Region as a hint subdirectory when MountPoint-like path is absent;
// prefers BLOCK local path via Region empty → ./blockstore.
func NewWithConfig(cfg block.Config) (*Store, error) {
	root := "./blockstore"
	if cfg.Region != "" {
		root = filepath.Join(root, cfg.Region)
	}
	return New(root)
}

func (s *Store) volPath(id string) string {
	return filepath.Join(s.root, "volumes", id+".json")
}

func (s *Store) snapPath(id string) string {
	return filepath.Join(s.root, "snapshots", id+".json")
}

func (s *Store) writeVol(vol *block.Volume) error {
	raw, err := json.MarshalIndent(vol, "", "  ")
	if err != nil {
		return errors.Internal("marshal volume", err)
	}
	return os.WriteFile(s.volPath(vol.ID), raw, 0o644)
}

func (s *Store) readVol(id string) (*block.Volume, error) {
	raw, err := os.ReadFile(s.volPath(id))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.NotFound("volume not found", err)
		}
		return nil, errors.Internal("read volume", err)
	}
	var vol block.Volume
	if err := json.Unmarshal(raw, &vol); err != nil {
		return nil, errors.Internal("unmarshal volume", err)
	}
	return &vol, nil
}

func (s *Store) writeSnap(snap *block.Snapshot) error {
	raw, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return errors.Internal("marshal snapshot", err)
	}
	return os.WriteFile(s.snapPath(snap.ID), raw, 0o644)
}

func (s *Store) readSnap(id string) (*block.Snapshot, error) {
	raw, err := os.ReadFile(s.snapPath(id))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.NotFound("snapshot not found", err)
		}
		return nil, errors.Internal("read snapshot", err)
	}
	var snap block.Snapshot
	if err := json.Unmarshal(raw, &snap); err != nil {
		return nil, errors.Internal("unmarshal snapshot", err)
	}
	return &snap, nil
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
		CreatedAt:        time.Now().UTC(),
		Tags:             opts.Tags,
		Attachments:      []block.Attachment{},
	}
	if vol.Tags == nil {
		vol.Tags = make(map[string]string)
	}
	if err := s.writeVol(vol); err != nil {
		return nil, err
	}
	return vol, nil
}

func (s *Store) GetVolume(ctx context.Context, volumeID string) (*block.Volume, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.readVol(volumeID)
}

func (s *Store) ListVolumes(ctx context.Context, opts block.ListOptions) (*block.ListResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries, err := os.ReadDir(filepath.Join(s.root, "volumes"))
	if err != nil {
		return nil, errors.Internal("list volumes", err)
	}
	result := &block.ListResult{Volumes: make([]*block.Volume, 0, len(entries))}
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		id := e.Name()[:len(e.Name())-5]
		vol, err := s.readVol(id)
		if err != nil {
			continue
		}
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
	if opts.Limit > 0 && len(result.Volumes) > opts.Limit {
		result.Volumes = result.Volumes[:opts.Limit]
		result.NextToken = "more"
	}
	return result, nil
}

func (s *Store) DeleteVolume(ctx context.Context, volumeID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	vol, err := s.readVol(volumeID)
	if err != nil {
		return err
	}
	if len(vol.Attachments) > 0 {
		return errors.Conflict("volume is attached to an instance", nil)
	}
	if err := os.Remove(s.volPath(volumeID)); err != nil {
		if os.IsNotExist(err) {
			return errors.NotFound("volume not found", err)
		}
		return errors.Internal("delete volume", err)
	}
	return nil
}

func (s *Store) ResizeVolume(ctx context.Context, volumeID string, opts block.ResizeVolumeOptions) (*block.Volume, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	vol, err := s.readVol(volumeID)
	if err != nil {
		return nil, err
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
	if err := s.writeVol(vol); err != nil {
		return nil, err
	}
	return vol, nil
}

func (s *Store) AttachVolume(ctx context.Context, opts block.AttachVolumeOptions) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	vol, err := s.readVol(opts.VolumeID)
	if err != nil {
		return err
	}
	for _, att := range vol.Attachments {
		if att.InstanceID == opts.InstanceID {
			return errors.Conflict("volume already attached to instance", nil)
		}
	}
	vol.Attachments = append(vol.Attachments, block.Attachment{
		InstanceID: opts.InstanceID,
		Device:     opts.Device,
		AttachedAt: time.Now().UTC(),
	})
	vol.State = block.VolumeStateInUse
	return s.writeVol(vol)
}

func (s *Store) DetachVolume(ctx context.Context, volumeID, instanceID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	vol, err := s.readVol(volumeID)
	if err != nil {
		return err
	}
	found := false
	newAtt := make([]block.Attachment, 0, len(vol.Attachments))
	for _, att := range vol.Attachments {
		if att.InstanceID == instanceID {
			found = true
			continue
		}
		newAtt = append(newAtt, att)
	}
	if !found {
		return errors.InvalidArgument("volume is not attached to instance", nil)
	}
	vol.Attachments = newAtt
	if len(vol.Attachments) == 0 {
		vol.State = block.VolumeStateAvailable
	}
	return s.writeVol(vol)
}

func (s *Store) CreateSnapshot(ctx context.Context, opts block.CreateSnapshotOptions) (*block.Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	vol, err := s.readVol(opts.VolumeID)
	if err != nil {
		return nil, err
	}
	snap := &block.Snapshot{
		ID:          uuid.NewString(),
		VolumeID:    opts.VolumeID,
		SizeGB:      vol.SizeGB,
		State:       "completed",
		CreatedAt:   time.Now().UTC(),
		Description: opts.Description,
		Tags:        opts.Tags,
	}
	if snap.Tags == nil {
		snap.Tags = make(map[string]string)
	}
	if err := s.writeSnap(snap); err != nil {
		return nil, err
	}
	return snap, nil
}

func (s *Store) GetSnapshot(ctx context.Context, snapshotID string) (*block.Snapshot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.readSnap(snapshotID)
}

func (s *Store) DeleteSnapshot(ctx context.Context, snapshotID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.Remove(s.snapPath(snapshotID)); err != nil {
		if os.IsNotExist(err) {
			return errors.NotFound("snapshot not found", err)
		}
		return errors.Internal("delete snapshot", err)
	}
	return nil
}

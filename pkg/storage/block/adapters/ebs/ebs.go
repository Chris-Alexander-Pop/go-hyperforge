package ebs

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/concurrency"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/storage/block"
	"github.com/google/uuid"
)

// Config configures the file-like EBS stub.
type Config struct {
	// Root is the local directory for volume/snapshot JSON (default ./ebsstore).
	Root string `env:"EBS_ROOT" env-default:"./ebsstore"`

	// Region is stamped into Tags when creating volumes.
	Region string `env:"BLOCK_REGION" env-default:"us-east-1"`

	// AvailabilityZone default AZ when CreateVolumeOptions.AvailabilityZone is empty.
	AvailabilityZone string `env:"BLOCK_AZ" env-default:"us-east-1a"`
}

// Store is a file-backed EBS-shaped VolumeStore.
type Store struct {
	root string
	cfg  Config
	mu   *concurrency.SmartRWMutex
}

var _ block.VolumeStore = (*Store)(nil)

// New creates an EBS stub store at cfg.Root.
func New(cfg Config) (*Store, error) {
	root := cfg.Root
	if root == "" {
		root = "./ebsstore"
	}
	if cfg.Region == "" {
		cfg.Region = "us-east-1"
	}
	if cfg.AvailabilityZone == "" {
		cfg.AvailabilityZone = cfg.Region + "a"
	}
	for _, sub := range []string{"volumes", "snapshots"} {
		if err := os.MkdirAll(filepath.Join(root, sub), 0o755); err != nil {
			return nil, errors.Wrap(err, "failed to create ebs store dir")
		}
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, errors.Wrap(err, "failed to resolve ebs root")
	}
	return &Store{
		root: abs,
		cfg:  cfg,
		mu:   concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "block-ebs"}),
	}, nil
}

// NewWithBlockConfig maps block.Config into the EBS stub.
func NewWithBlockConfig(bc block.Config) (*Store, error) {
	root := "./ebsstore"
	if bc.Region != "" {
		root = filepath.Join(root, bc.Region)
	}
	return New(Config{
		Root:             root,
		Region:           bc.Region,
		AvailabilityZone: bc.AvailabilityZone,
	})
}

func (s *Store) volPath(id string) string {
	return filepath.Join(s.root, "volumes", id+".json")
}

func (s *Store) snapPath(id string) string {
	return filepath.Join(s.root, "snapshots", id+".json")
}

func volID() string {
	return "vol-" + strings.ReplaceAll(uuid.NewString(), "-", "")[:17]
}

func snapID() string {
	return "snap-" + strings.ReplaceAll(uuid.NewString(), "-", "")[:16]
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
			return nil, block.ErrVolumeNotFound
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
			return nil, block.ErrSnapshotNotFound
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
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()

	tags := opts.Tags
	if tags == nil {
		tags = map[string]string{}
	}
	if opts.SnapshotID != "" {
		snap, err := s.readSnap(opts.SnapshotID)
		if err != nil {
			return nil, err
		}
		if opts.SizeGB <= 0 {
			opts.SizeGB = snap.SizeGB
		}
		if opts.SizeGB < snap.SizeGB {
			return nil, block.ErrSizeTooSmall
		}
		tags["ebs.amazonaws.com/source-snapshot"] = opts.SnapshotID
	}
	if opts.SizeGB <= 0 {
		return nil, block.ErrInvalidSize
	}
	volType := opts.VolumeType
	if volType == "" {
		volType = block.VolumeTypeSSD
	}
	az := opts.AvailabilityZone
	if az == "" {
		az = s.cfg.AvailabilityZone
	}
	tags["ebs.amazonaws.com/region"] = s.cfg.Region
	vol := &block.Volume{
		ID:               volID(),
		Name:             opts.Name,
		SizeGB:           opts.SizeGB,
		State:            block.VolumeStateAvailable,
		VolumeType:       volType,
		AvailabilityZone: az,
		Encrypted:        opts.Encrypted,
		IOPS:             opts.IOPS,
		Throughput:       opts.Throughput,
		CreatedAt:        time.Now().UTC(),
		Tags:             tags,
		Attachments:      []block.Attachment{},
	}
	if err := s.writeVol(vol); err != nil {
		return nil, err
	}
	return vol, nil
}

func (s *Store) GetVolume(ctx context.Context, volumeID string) (*block.Volume, error) {
	_ = ctx
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.readVol(volumeID)
}

func (s *Store) ListVolumes(ctx context.Context, opts block.ListOptions) (*block.ListResult, error) {
	_ = ctx
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
		id := strings.TrimSuffix(e.Name(), ".json")
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
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	vol, err := s.readVol(volumeID)
	if err != nil {
		return err
	}
	if len(vol.Attachments) > 0 {
		return block.ErrVolumeInUse
	}
	if err := os.Remove(s.volPath(volumeID)); err != nil {
		if os.IsNotExist(err) {
			return block.ErrVolumeNotFound
		}
		return errors.Internal("delete volume", err)
	}
	return nil
}

func (s *Store) ResizeVolume(ctx context.Context, volumeID string, opts block.ResizeVolumeOptions) (*block.Volume, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	vol, err := s.readVol(volumeID)
	if err != nil {
		return nil, err
	}
	if opts.NewSizeGB > 0 {
		if opts.NewSizeGB < vol.SizeGB {
			return nil, block.ErrSizeTooSmall
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
	_ = ctx
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
	device := opts.Device
	if device == "" {
		device = fmt.Sprintf("/dev/xvd%c", 'f'+len(vol.Attachments))
	}
	vol.Attachments = append(vol.Attachments, block.Attachment{
		InstanceID: opts.InstanceID,
		Device:     device,
		AttachedAt: time.Now().UTC(),
	})
	vol.State = block.VolumeStateInUse
	return s.writeVol(vol)
}

func (s *Store) DetachVolume(ctx context.Context, volumeID, instanceID string) error {
	_ = ctx
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
		return block.ErrVolumeNotAttached
	}
	vol.Attachments = newAtt
	if len(vol.Attachments) == 0 {
		vol.State = block.VolumeStateAvailable
	}
	return s.writeVol(vol)
}

func (s *Store) CreateSnapshot(ctx context.Context, opts block.CreateSnapshotOptions) (*block.Snapshot, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	vol, err := s.readVol(opts.VolumeID)
	if err != nil {
		return nil, err
	}
	snap := &block.Snapshot{
		ID:          snapID(),
		VolumeID:    opts.VolumeID,
		SizeGB:      vol.SizeGB,
		State:       "completed",
		CreatedAt:   time.Now().UTC(),
		Description: opts.Description,
		Tags:        opts.Tags,
	}
	if snap.Tags == nil {
		snap.Tags = map[string]string{}
	}
	if err := s.writeSnap(snap); err != nil {
		return nil, err
	}
	return snap, nil
}

func (s *Store) GetSnapshot(ctx context.Context, snapshotID string) (*block.Snapshot, error) {
	_ = ctx
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.readSnap(snapshotID)
}

func (s *Store) DeleteSnapshot(ctx context.Context, snapshotID string) error {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.Remove(s.snapPath(snapshotID)); err != nil {
		if os.IsNotExist(err) {
			return block.ErrSnapshotNotFound
		}
		return errors.Internal("delete snapshot", err)
	}
	return nil
}

// ListSnapshots returns snapshots under the store root (optional convenience beyond VolumeStore).
func (s *Store) ListSnapshots(ctx context.Context) ([]*block.Snapshot, error) {
	_ = ctx
	s.mu.RLock()
	defer s.mu.RUnlock()
	entries, err := os.ReadDir(filepath.Join(s.root, "snapshots"))
	if err != nil {
		return nil, errors.Internal("list snapshots", err)
	}
	out := make([]*block.Snapshot, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		id := strings.TrimSuffix(e.Name(), ".json")
		snap, err := s.readSnap(id)
		if err != nil {
			continue
		}
		out = append(out, snap)
	}
	return out, nil
}

// Package lvm provides a local/LVM-shaped VolumeController for tests and dev.
package lvm

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/concurrency"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/storage/controller"
	"github.com/google/uuid"
)

// Config configures the local LVM-shaped controller.
type Config struct {
	// Root is the directory for volume files and metadata (default ./lvmstore).
	Root string `env:"LVM_ROOT" env-default:"./lvmstore"`

	// VolumeGroup is stamped into volume tags (cosmetic; no real VG).
	VolumeGroup string `env:"LVM_VG" env-default:"hyperforge"`
}

// Controller implements controller.VolumeController with sparse files + JSON meta.
type Controller struct {
	root string
	vg   string
	mu   *concurrency.SmartRWMutex
}

var _ controller.VolumeController = (*Controller)(nil)

// New creates a local LVM-shaped volume controller.
func New(cfg Config) (*Controller, error) {
	root := cfg.Root
	if root == "" {
		root = "./lvmstore"
	}
	vg := cfg.VolumeGroup
	if vg == "" {
		vg = "hyperforge"
	}
	for _, sub := range []string{"volumes", "meta"} {
		if err := os.MkdirAll(filepath.Join(root, sub), 0o755); err != nil {
			return nil, errors.Wrap(err, "failed to create lvm store dir")
		}
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, errors.Wrap(err, "failed to resolve lvm root")
	}
	return &Controller{
		root: abs,
		vg:   vg,
		mu:   concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "controller-lvm"}),
	}, nil
}

func (c *Controller) metaPath(id string) string {
	return filepath.Join(c.root, "meta", id+".json")
}

func (c *Controller) dataPath(id string) string {
	return filepath.Join(c.root, "volumes", id+".img")
}

func (c *Controller) writeMeta(vol *controller.Volume) error {
	raw, err := json.MarshalIndent(vol, "", "  ")
	if err != nil {
		return errors.Internal("marshal volume", err)
	}
	return os.WriteFile(c.metaPath(vol.ID), raw, 0o644)
}

func (c *Controller) readMeta(id string) (*controller.Volume, error) {
	raw, err := os.ReadFile(c.metaPath(id))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, controller.ErrVolumeNotFound
		}
		return nil, errors.Internal("read volume meta", err)
	}
	var vol controller.Volume
	if err := json.Unmarshal(raw, &vol); err != nil {
		return nil, errors.Internal("unmarshal volume meta", err)
	}
	return &vol, nil
}

func (c *Controller) CreateVolume(ctx context.Context, spec controller.VolumeSpec) (string, error) {
	_ = ctx
	c.mu.Lock()
	defer c.mu.Unlock()

	if spec.SizeGB <= 0 {
		return "", controller.ErrInvalidSize
	}
	id := "lv-" + uuid.NewString()[:12]
	tags := spec.Tags
	if tags == nil {
		tags = map[string]string{}
	}
	tags["lvm.volume_group"] = c.vg
	tags["lvm.path"] = c.dataPath(id)
	if spec.SnapshotID != "" {
		tags["lvm.source_snapshot"] = spec.SnapshotID
	}

	// Sparse file: truncate to SizeGB without allocating full disk (enough for tests).
	f, err := os.OpenFile(c.dataPath(id), os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return "", errors.Internal("create volume file", err)
	}
	size := int64(spec.SizeGB) * 1024 * 1024 // use MB units in tests to keep files small
	if err := f.Truncate(size); err != nil {
		_ = f.Close()
		_ = os.Remove(c.dataPath(id))
		return "", errors.Internal("truncate volume file", err)
	}
	_ = f.Close()

	vol := &controller.Volume{
		ID:        id,
		Name:      spec.Name,
		SizeGB:    spec.SizeGB,
		Type:      spec.Type,
		Zone:      spec.Zone,
		Status:    controller.VolumeStatusAvailable,
		CreatedAt: time.Now().UTC(),
		Tags:      tags,
	}
	if vol.Type == "" {
		vol.Type = "lvm"
	}
	if err := c.writeMeta(vol); err != nil {
		_ = os.Remove(c.dataPath(id))
		return "", err
	}
	return id, nil
}

func (c *Controller) DeleteVolume(ctx context.Context, volumeID string) error {
	_ = ctx
	c.mu.Lock()
	defer c.mu.Unlock()

	vol, err := c.readMeta(volumeID)
	if err != nil {
		return err
	}
	if vol.Status == controller.VolumeStatusAttached {
		return controller.ErrVolumeAttached
	}
	_ = os.Remove(c.dataPath(volumeID))
	if err := os.Remove(c.metaPath(volumeID)); err != nil && !os.IsNotExist(err) {
		return errors.Internal("delete volume meta", err)
	}
	return nil
}

func (c *Controller) AttachVolume(ctx context.Context, volumeID string, nodeID string) error {
	_ = ctx
	c.mu.Lock()
	defer c.mu.Unlock()

	vol, err := c.readMeta(volumeID)
	if err != nil {
		return err
	}
	if vol.Status == controller.VolumeStatusAttached {
		if vol.AttachedTo == nodeID {
			return nil
		}
		return controller.ErrVolumeAttached
	}
	vol.Status = controller.VolumeStatusAttached
	vol.AttachedTo = nodeID
	if vol.Tags == nil {
		vol.Tags = map[string]string{}
	}
	vol.Tags["lvm.device"] = "/dev/mapper/" + c.vg + "-" + volumeID
	return c.writeMeta(vol)
}

func (c *Controller) DetachVolume(ctx context.Context, volumeID string) error {
	_ = ctx
	c.mu.Lock()
	defer c.mu.Unlock()

	vol, err := c.readMeta(volumeID)
	if err != nil {
		return err
	}
	vol.Status = controller.VolumeStatusAvailable
	vol.AttachedTo = ""
	if vol.Tags != nil {
		delete(vol.Tags, "lvm.device")
	}
	return c.writeMeta(vol)
}

func (c *Controller) ResizeVolume(ctx context.Context, volumeID string, newSizeGB int) error {
	_ = ctx
	c.mu.Lock()
	defer c.mu.Unlock()

	vol, err := c.readMeta(volumeID)
	if err != nil {
		return err
	}
	if newSizeGB < vol.SizeGB {
		return controller.ErrInvalidSize
	}
	f, err := os.OpenFile(c.dataPath(volumeID), os.O_RDWR, 0o644)
	if err != nil {
		return errors.Internal("open volume for resize", err)
	}
	defer f.Close()
	if err := f.Truncate(int64(newSizeGB) * 1024 * 1024); err != nil {
		return errors.Internal("resize volume file", err)
	}
	vol.SizeGB = newSizeGB
	return c.writeMeta(vol)
}

func (c *Controller) GetVolume(ctx context.Context, volumeID string) (*controller.Volume, error) {
	_ = ctx
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.readMeta(volumeID)
}

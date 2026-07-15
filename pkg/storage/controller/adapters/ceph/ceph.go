package ceph

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/concurrency"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/storage/controller"
	"github.com/google/uuid"
)

const defaultRBDPool = "rbd"

// RBDImage is a Ceph RBD image as seen by the injectable client.
type RBDImage struct {
	ID         string
	Name       string
	SizeGB     int
	Pool       string
	Status     controller.VolumeStatus
	AttachedTo string
	CreatedAt  time.Time
	Tags       map[string]string
}

// CreateImageRequest is the input for RBDClient.CreateImage.
type CreateImageRequest struct {
	Name       string
	SizeGB     int
	Pool       string
	SnapshotID string
	Tags       map[string]string
}

// RBDClient is an injectable Ceph RBD-shaped API (HTTP gateway or test double).
// It is intentionally not librados — callers supply their own transport.
type RBDClient interface {
	CreateImage(ctx context.Context, req CreateImageRequest) (*RBDImage, error)
	DeleteImage(ctx context.Context, imageID string) error
	MapImage(ctx context.Context, imageID, nodeID string) error
	UnmapImage(ctx context.Context, imageID string) error
	ResizeImage(ctx context.Context, imageID string, newSizeGB int) error
	GetImage(ctx context.Context, imageID string) (*RBDImage, error)
}

// Config configures the Ceph RBD-shaped controller.
type Config struct {
	// Pool is the default RBD pool (stamped into volume zone/tags).
	Pool string `env:"CEPH_POOL" env-default:"rbd"`

	// Client is required; use NewMemoryRBDClient for tests.
	Client RBDClient
}

// Controller implements controller.VolumeController via an injectable RBDClient.
type Controller struct {
	client RBDClient
	pool   string
	mu     *concurrency.SmartRWMutex
}

var _ controller.VolumeController = (*Controller)(nil)

// New creates a Ceph RBD-shaped volume controller.
func New(cfg Config) (*Controller, error) {
	if cfg.Client == nil {
		return nil, errors.InvalidArgument("ceph rbd client is required", nil)
	}
	pool := cfg.Pool
	if pool == "" {
		pool = defaultRBDPool
	}
	return &Controller{
		client: cfg.Client,
		pool:   pool,
		mu:     concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "controller-ceph"}),
	}, nil
}

func (c *Controller) CreateVolume(ctx context.Context, spec controller.VolumeSpec) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if spec.SizeGB <= 0 {
		return "", controller.ErrInvalidSize
	}
	pool := spec.Zone
	if pool == "" {
		pool = c.pool
	}
	img, err := c.client.CreateImage(ctx, CreateImageRequest{
		Name:       spec.Name,
		SizeGB:     spec.SizeGB,
		Pool:       pool,
		SnapshotID: spec.SnapshotID,
		Tags:       copyTags(spec.Tags),
	})
	if err != nil {
		return "", errors.Unavailable("ceph CreateImage failed", err)
	}
	return img.ID, nil
}

func (c *Controller) DeleteVolume(ctx context.Context, volumeID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	img, err := c.client.GetImage(ctx, volumeID)
	if err != nil {
		return mapClientErr(err)
	}
	if img.Status == controller.VolumeStatusAttached {
		return controller.ErrVolumeAttached
	}
	if err := c.client.DeleteImage(ctx, volumeID); err != nil {
		return errors.Unavailable("ceph DeleteImage failed", err)
	}
	return nil
}

func (c *Controller) AttachVolume(ctx context.Context, volumeID string, nodeID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	img, err := c.client.GetImage(ctx, volumeID)
	if err != nil {
		return mapClientErr(err)
	}
	if img.Status == controller.VolumeStatusAttached {
		if img.AttachedTo == nodeID {
			return nil
		}
		return controller.ErrVolumeAttached
	}
	if err := c.client.MapImage(ctx, volumeID, nodeID); err != nil {
		return errors.Unavailable("ceph MapImage failed", err)
	}
	return nil
}

func (c *Controller) DetachVolume(ctx context.Context, volumeID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, err := c.client.GetImage(ctx, volumeID); err != nil {
		return mapClientErr(err)
	}
	if err := c.client.UnmapImage(ctx, volumeID); err != nil {
		return errors.Unavailable("ceph UnmapImage failed", err)
	}
	return nil
}

func (c *Controller) ResizeVolume(ctx context.Context, volumeID string, newSizeGB int) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	img, err := c.client.GetImage(ctx, volumeID)
	if err != nil {
		return mapClientErr(err)
	}
	if newSizeGB < img.SizeGB {
		return controller.ErrInvalidSize
	}
	if err := c.client.ResizeImage(ctx, volumeID, newSizeGB); err != nil {
		return errors.Unavailable("ceph ResizeImage failed", err)
	}
	return nil
}

func (c *Controller) GetVolume(ctx context.Context, volumeID string) (*controller.Volume, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	img, err := c.client.GetImage(ctx, volumeID)
	if err != nil {
		return nil, mapClientErr(err)
	}
	return imageToVolume(img), nil
}

func imageToVolume(img *RBDImage) *controller.Volume {
	tags := copyTags(img.Tags)
	if tags == nil {
		tags = map[string]string{}
	}
	tags["ceph.pool"] = img.Pool
	volType := defaultRBDPool
	if t, ok := tags["type"]; ok && t != "" {
		volType = t
	}
	return &controller.Volume{
		ID:         img.ID,
		Name:       img.Name,
		SizeGB:     img.SizeGB,
		Type:       volType,
		Zone:       img.Pool,
		Status:     img.Status,
		AttachedTo: img.AttachedTo,
		CreatedAt:  img.CreatedAt,
		Tags:       tags,
	}
}

func mapClientErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, controller.ErrVolumeNotFound) || errors.IsCode(err, errors.CodeNotFound) {
		return controller.ErrVolumeNotFound
	}
	return errors.Unavailable("ceph client error", err)
}

func copyTags(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

// MemoryRBDClient is an in-process RBDClient for unit tests.
type MemoryRBDClient struct {
	mu     *concurrency.SmartRWMutex
	images map[string]*RBDImage
	pool   string
}

// NewMemoryRBDClient creates a test double RBD client.
func NewMemoryRBDClient(pool string) *MemoryRBDClient {
	if pool == "" {
		pool = defaultRBDPool
	}
	return &MemoryRBDClient{
		mu:     concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "ceph-memory-rbd"}),
		images: make(map[string]*RBDImage),
		pool:   pool,
	}
}

func (m *MemoryRBDClient) CreateImage(_ context.Context, req CreateImageRequest) (*RBDImage, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if req.SizeGB <= 0 {
		return nil, controller.ErrInvalidSize
	}
	pool := req.Pool
	if pool == "" {
		pool = m.pool
	}
	id := "rbd-" + uuid.NewString()[:12]
	tags := copyTags(req.Tags)
	if tags == nil {
		tags = map[string]string{}
	}
	if req.SnapshotID != "" {
		tags["ceph.source_snapshot"] = req.SnapshotID
	}
	img := &RBDImage{
		ID:        id,
		Name:      req.Name,
		SizeGB:    req.SizeGB,
		Pool:      pool,
		Status:    controller.VolumeStatusAvailable,
		CreatedAt: time.Now().UTC(),
		Tags:      tags,
	}
	m.images[id] = img
	cp := *img
	cp.Tags = copyTags(img.Tags)
	return &cp, nil
}

func (m *MemoryRBDClient) DeleteImage(_ context.Context, imageID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	img, ok := m.images[imageID]
	if !ok {
		return controller.ErrVolumeNotFound
	}
	if img.Status == controller.VolumeStatusAttached {
		return controller.ErrVolumeAttached
	}
	delete(m.images, imageID)
	return nil
}

func (m *MemoryRBDClient) MapImage(_ context.Context, imageID, nodeID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	img, ok := m.images[imageID]
	if !ok {
		return controller.ErrVolumeNotFound
	}
	if img.Status == controller.VolumeStatusAttached {
		if img.AttachedTo == nodeID {
			return nil
		}
		return controller.ErrVolumeAttached
	}
	img.Status = controller.VolumeStatusAttached
	img.AttachedTo = nodeID
	if img.Tags == nil {
		img.Tags = map[string]string{}
	}
	img.Tags["ceph.device"] = "/dev/rbd/" + img.Pool + "/" + imageID
	return nil
}

func (m *MemoryRBDClient) UnmapImage(_ context.Context, imageID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	img, ok := m.images[imageID]
	if !ok {
		return controller.ErrVolumeNotFound
	}
	img.Status = controller.VolumeStatusAvailable
	img.AttachedTo = ""
	if img.Tags != nil {
		delete(img.Tags, "ceph.device")
	}
	return nil
}

func (m *MemoryRBDClient) ResizeImage(_ context.Context, imageID string, newSizeGB int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	img, ok := m.images[imageID]
	if !ok {
		return controller.ErrVolumeNotFound
	}
	if newSizeGB < img.SizeGB {
		return controller.ErrInvalidSize
	}
	img.SizeGB = newSizeGB
	return nil
}

func (m *MemoryRBDClient) GetImage(_ context.Context, imageID string) (*RBDImage, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	img, ok := m.images[imageID]
	if !ok {
		return nil, controller.ErrVolumeNotFound
	}
	cp := *img
	cp.Tags = copyTags(img.Tags)
	return &cp, nil
}

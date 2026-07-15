package csi

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/concurrency"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/storage/controller"
	"github.com/google/uuid"
)

// Volume is a CSI volume as returned by the injectable controller API.
type Volume struct {
	ID         string
	Name       string
	SizeGB     int
	Type       string
	Zone       string
	Status     controller.VolumeStatus
	AttachedTo string
	CreatedAt  time.Time
	Tags       map[string]string
}

// CreateVolumeRequest is the input for CSIControllerAPI.CreateVolume.
type CreateVolumeRequest struct {
	Name       string
	SizeGB     int
	Type       string
	Zone       string
	SnapshotID string
	Tags       map[string]string
}

// CSIControllerAPI is an injectable CSI Controller-shaped surface.
// Real deployments may wrap a CSI gRPC ControllerClient behind this interface.
type CSIControllerAPI interface {
	CreateVolume(ctx context.Context, req CreateVolumeRequest) (*Volume, error)
	DeleteVolume(ctx context.Context, volumeID string) error
	ControllerPublishVolume(ctx context.Context, volumeID, nodeID string) error
	ControllerUnpublishVolume(ctx context.Context, volumeID string) error
	ControllerExpandVolume(ctx context.Context, volumeID string, newSizeGB int) error
	ControllerGetVolume(ctx context.Context, volumeID string) (*Volume, error)
}

// Config configures the CSI-shaped controller.
type Config struct {
	// DriverName is stamped into volume tags (e.g. "csi.hostpath").
	DriverName string `env:"CSI_DRIVER" env-default:"csi.hyperforge"`

	// Client is required; use NewMemoryCSIAPI for tests.
	Client CSIControllerAPI
}

// Controller implements controller.VolumeController via CSIControllerAPI.
type Controller struct {
	client     CSIControllerAPI
	driverName string
	mu         *concurrency.SmartRWMutex
}

var _ controller.VolumeController = (*Controller)(nil)

// New creates a CSI-shaped volume controller.
func New(cfg Config) (*Controller, error) {
	if cfg.Client == nil {
		return nil, errors.InvalidArgument("csi controller api is required", nil)
	}
	name := cfg.DriverName
	if name == "" {
		name = "csi.hyperforge"
	}
	return &Controller{
		client:     cfg.Client,
		driverName: name,
		mu:         concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "controller-csi"}),
	}, nil
}

func (c *Controller) CreateVolume(ctx context.Context, spec controller.VolumeSpec) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if spec.SizeGB <= 0 {
		return "", controller.ErrInvalidSize
	}
	tags := copyTags(spec.Tags)
	if tags == nil {
		tags = map[string]string{}
	}
	tags["csi.driver"] = c.driverName
	vol, err := c.client.CreateVolume(ctx, CreateVolumeRequest{
		Name:       spec.Name,
		SizeGB:     spec.SizeGB,
		Type:       spec.Type,
		Zone:       spec.Zone,
		SnapshotID: spec.SnapshotID,
		Tags:       tags,
	})
	if err != nil {
		return "", errors.Unavailable("csi CreateVolume failed", err)
	}
	return vol.ID, nil
}

func (c *Controller) DeleteVolume(ctx context.Context, volumeID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	vol, err := c.client.ControllerGetVolume(ctx, volumeID)
	if err != nil {
		return mapClientErr(err)
	}
	if vol.Status == controller.VolumeStatusAttached {
		return controller.ErrVolumeAttached
	}
	if err := c.client.DeleteVolume(ctx, volumeID); err != nil {
		return errors.Unavailable("csi DeleteVolume failed", err)
	}
	return nil
}

// AttachVolume maps to CSI ControllerPublishVolume.
func (c *Controller) AttachVolume(ctx context.Context, volumeID string, nodeID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	vol, err := c.client.ControllerGetVolume(ctx, volumeID)
	if err != nil {
		return mapClientErr(err)
	}
	if vol.Status == controller.VolumeStatusAttached {
		if vol.AttachedTo == nodeID {
			return nil
		}
		return controller.ErrVolumeAttached
	}
	if err := c.client.ControllerPublishVolume(ctx, volumeID, nodeID); err != nil {
		return errors.Unavailable("csi ControllerPublishVolume failed", err)
	}
	return nil
}

// DetachVolume maps to CSI ControllerUnpublishVolume.
func (c *Controller) DetachVolume(ctx context.Context, volumeID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, err := c.client.ControllerGetVolume(ctx, volumeID); err != nil {
		return mapClientErr(err)
	}
	if err := c.client.ControllerUnpublishVolume(ctx, volumeID); err != nil {
		return errors.Unavailable("csi ControllerUnpublishVolume failed", err)
	}
	return nil
}

func (c *Controller) ResizeVolume(ctx context.Context, volumeID string, newSizeGB int) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	vol, err := c.client.ControllerGetVolume(ctx, volumeID)
	if err != nil {
		return mapClientErr(err)
	}
	if newSizeGB < vol.SizeGB {
		return controller.ErrInvalidSize
	}
	if err := c.client.ControllerExpandVolume(ctx, volumeID, newSizeGB); err != nil {
		return errors.Unavailable("csi ControllerExpandVolume failed", err)
	}
	return nil
}

func (c *Controller) GetVolume(ctx context.Context, volumeID string) (*controller.Volume, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	vol, err := c.client.ControllerGetVolume(ctx, volumeID)
	if err != nil {
		return nil, mapClientErr(err)
	}
	return toControllerVolume(vol), nil
}

func toControllerVolume(v *Volume) *controller.Volume {
	typ := v.Type
	if typ == "" {
		typ = "csi"
	}
	return &controller.Volume{
		ID:         v.ID,
		Name:       v.Name,
		SizeGB:     v.SizeGB,
		Type:       typ,
		Zone:       v.Zone,
		Status:     v.Status,
		AttachedTo: v.AttachedTo,
		CreatedAt:  v.CreatedAt,
		Tags:       copyTags(v.Tags),
	}
}

func mapClientErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, controller.ErrVolumeNotFound) || errors.IsCode(err, errors.CodeNotFound) {
		return controller.ErrVolumeNotFound
	}
	return errors.Unavailable("csi client error", err)
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

// MemoryCSIAPI is an in-process CSIControllerAPI for unit tests.
type MemoryCSIAPI struct {
	mu      *concurrency.SmartRWMutex
	volumes map[string]*Volume
	driver  string
}

// NewMemoryCSIAPI creates a fake CSI controller API.
func NewMemoryCSIAPI(driverName string) *MemoryCSIAPI {
	if driverName == "" {
		driverName = "csi.hyperforge"
	}
	return &MemoryCSIAPI{
		mu:      concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "csi-memory-api"}),
		volumes: make(map[string]*Volume),
		driver:  driverName,
	}
}

func (m *MemoryCSIAPI) CreateVolume(_ context.Context, req CreateVolumeRequest) (*Volume, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if req.SizeGB <= 0 {
		return nil, controller.ErrInvalidSize
	}
	id := "pvc-" + uuid.NewString()[:12]
	tags := copyTags(req.Tags)
	if tags == nil {
		tags = map[string]string{}
	}
	if _, ok := tags["csi.driver"]; !ok {
		tags["csi.driver"] = m.driver
	}
	if req.SnapshotID != "" {
		tags["csi.source_snapshot"] = req.SnapshotID
	}
	typ := req.Type
	if typ == "" {
		typ = "csi"
	}
	vol := &Volume{
		ID:        id,
		Name:      req.Name,
		SizeGB:    req.SizeGB,
		Type:      typ,
		Zone:      req.Zone,
		Status:    controller.VolumeStatusAvailable,
		CreatedAt: time.Now().UTC(),
		Tags:      tags,
	}
	m.volumes[id] = vol
	return cloneVolume(vol), nil
}

func (m *MemoryCSIAPI) DeleteVolume(_ context.Context, volumeID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	vol, ok := m.volumes[volumeID]
	if !ok {
		return controller.ErrVolumeNotFound
	}
	if vol.Status == controller.VolumeStatusAttached {
		return controller.ErrVolumeAttached
	}
	delete(m.volumes, volumeID)
	return nil
}

func (m *MemoryCSIAPI) ControllerPublishVolume(_ context.Context, volumeID, nodeID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	vol, ok := m.volumes[volumeID]
	if !ok {
		return controller.ErrVolumeNotFound
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
	vol.Tags["csi.published_node"] = nodeID
	return nil
}

func (m *MemoryCSIAPI) ControllerUnpublishVolume(_ context.Context, volumeID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	vol, ok := m.volumes[volumeID]
	if !ok {
		return controller.ErrVolumeNotFound
	}
	vol.Status = controller.VolumeStatusAvailable
	vol.AttachedTo = ""
	if vol.Tags != nil {
		delete(vol.Tags, "csi.published_node")
	}
	return nil
}

func (m *MemoryCSIAPI) ControllerExpandVolume(_ context.Context, volumeID string, newSizeGB int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	vol, ok := m.volumes[volumeID]
	if !ok {
		return controller.ErrVolumeNotFound
	}
	if newSizeGB < vol.SizeGB {
		return controller.ErrInvalidSize
	}
	vol.SizeGB = newSizeGB
	return nil
}

func (m *MemoryCSIAPI) ControllerGetVolume(_ context.Context, volumeID string) (*Volume, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	vol, ok := m.volumes[volumeID]
	if !ok {
		return nil, controller.ErrVolumeNotFound
	}
	return cloneVolume(vol), nil
}

func cloneVolume(v *Volume) *Volume {
	cp := *v
	cp.Tags = copyTags(v.Tags)
	return &cp
}

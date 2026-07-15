package memory

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/concurrency"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/storage/controller"
	"github.com/google/uuid"
)

// MemoryVolumeController is an in-memory implementation of VolumeController.
type MemoryVolumeController struct {
	volumes map[string]*controller.Volume
	mu      *concurrency.SmartRWMutex
}

// New creates a new MEMoryVolumeController.
func New() *MemoryVolumeController {
	return &MemoryVolumeController{
		volumes: make(map[string]*controller.Volume),
		mu: concurrency.NewSmartRWMutex(concurrency.MutexConfig{
			Name: "memory-volume-controller",
		}),
	}
}

func (c *MemoryVolumeController) CreateVolume(ctx context.Context, spec controller.VolumeSpec) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	id := uuid.NewString()
	vol := &controller.Volume{
		ID:        id,
		Name:      spec.Name,
		SizeGB:    spec.SizeGB,
		Type:      spec.Type,
		Zone:      spec.Zone,
		Status:    controller.VolumeStatusAvailable,
		CreatedAt: time.Now(),
		Tags:      spec.Tags,
	}

	c.volumes[id] = vol
	return id, nil
}

func (c *MemoryVolumeController) DeleteVolume(ctx context.Context, volumeID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	vol, ok := c.volumes[volumeID]
	if !ok {
		return controller.ErrVolumeNotFound
	}

	if vol.Status == controller.VolumeStatusAttached {
		return controller.ErrVolumeAttached
	}

	delete(c.volumes, volumeID)
	return nil
}

func (c *MemoryVolumeController) AttachVolume(ctx context.Context, volumeID string, nodeID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	vol, ok := c.volumes[volumeID]
	if !ok {
		return controller.ErrVolumeNotFound
	}

	if vol.Status == controller.VolumeStatusAttached {
		// If attached to same node, redundant success
		if vol.AttachedTo == nodeID {
			return nil
		}
		return controller.ErrVolumeAttached
	}

	vol.Status = controller.VolumeStatusAttached
	vol.AttachedTo = nodeID
	return nil
}

func (c *MemoryVolumeController) DetachVolume(ctx context.Context, volumeID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	vol, ok := c.volumes[volumeID]
	if !ok {
		return controller.ErrVolumeNotFound
	}

	vol.Status = controller.VolumeStatusAvailable
	vol.AttachedTo = ""
	return nil
}

func (c *MemoryVolumeController) ResizeVolume(ctx context.Context, volumeID string, newSizeGB int) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	vol, ok := c.volumes[volumeID]
	if !ok {
		return controller.ErrVolumeNotFound
	}

	if newSizeGB < vol.SizeGB {
		return controller.ErrInvalidSize
	}

	vol.SizeGB = newSizeGB
	return nil
}

func (c *MemoryVolumeController) GetVolume(ctx context.Context, volumeID string) (*controller.Volume, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	vol, ok := c.volumes[volumeID]
	if !ok {
		return nil, controller.ErrVolumeNotFound
	}

	v := *vol
	return &v, nil
}

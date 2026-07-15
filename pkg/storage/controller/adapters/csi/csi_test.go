package csi_test

import (
	"context"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/storage/controller"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/storage/controller/adapters/csi"
	"github.com/stretchr/testify/require"
)

func TestCSIControllerLifecycle(t *testing.T) {
	api := csi.NewMemoryCSIAPI("csi.hostpath")
	c, err := csi.New(csi.Config{DriverName: "csi.hostpath", Client: api})
	require.NoError(t, err)

	ctx := context.Background()
	id, err := c.CreateVolume(ctx, controller.VolumeSpec{
		Name:   "pvc-data",
		SizeGB: 8,
		Type:   "ssd",
		Zone:   "zone-a",
		Tags:   map[string]string{"app": "db"},
	})
	require.NoError(t, err)
	require.Contains(t, id, "pvc-")

	vol, err := c.GetVolume(ctx, id)
	require.NoError(t, err)
	require.Equal(t, controller.VolumeStatusAvailable, vol.Status)
	require.Equal(t, "csi.hostpath", vol.Tags["csi.driver"])
	require.Equal(t, "zone-a", vol.Zone)
	require.Equal(t, "db", vol.Tags["app"])

	// Attach maps to ControllerPublish
	require.NoError(t, c.AttachVolume(ctx, id, "node-a"))
	vol, err = c.GetVolume(ctx, id)
	require.NoError(t, err)
	require.Equal(t, controller.VolumeStatusAttached, vol.Status)
	require.Equal(t, "node-a", vol.AttachedTo)
	require.Equal(t, "node-a", vol.Tags["csi.published_node"])

	require.NoError(t, c.AttachVolume(ctx, id, "node-a"))
	require.ErrorIs(t, c.AttachVolume(ctx, id, "node-b"), controller.ErrVolumeAttached)
	require.ErrorIs(t, c.DeleteVolume(ctx, id), controller.ErrVolumeAttached)

	// Detach maps to ControllerUnpublish
	require.NoError(t, c.DetachVolume(ctx, id))
	require.NoError(t, c.ResizeVolume(ctx, id, 16))
	vol, err = c.GetVolume(ctx, id)
	require.NoError(t, err)
	require.Equal(t, 16, vol.SizeGB)
	require.Equal(t, controller.VolumeStatusAvailable, vol.Status)
	_, ok := vol.Tags["csi.published_node"]
	require.False(t, ok)

	require.ErrorIs(t, c.ResizeVolume(ctx, id, 1), controller.ErrInvalidSize)
	require.NoError(t, c.DeleteVolume(ctx, id))
	_, err = c.GetVolume(ctx, id)
	require.ErrorIs(t, err, controller.ErrVolumeNotFound)
}

func TestCSINewRequiresClient(t *testing.T) {
	_, err := csi.New(csi.Config{})
	require.Error(t, err)
}

func TestCSIInvalidSize(t *testing.T) {
	c, err := csi.New(csi.Config{Client: csi.NewMemoryCSIAPI("")})
	require.NoError(t, err)
	_, err = c.CreateVolume(context.Background(), controller.VolumeSpec{Name: "x", SizeGB: -1})
	require.ErrorIs(t, err, controller.ErrInvalidSize)
}

func TestCSICreateFromSnapshot(t *testing.T) {
	c, err := csi.New(csi.Config{DriverName: "csi.test", Client: csi.NewMemoryCSIAPI("csi.test")})
	require.NoError(t, err)
	id, err := c.CreateVolume(context.Background(), controller.VolumeSpec{
		Name: "from-snap", SizeGB: 4, SnapshotID: "snap-xyz",
	})
	require.NoError(t, err)
	vol, err := c.GetVolume(context.Background(), id)
	require.NoError(t, err)
	require.Equal(t, "snap-xyz", vol.Tags["csi.source_snapshot"])
	require.Equal(t, "csi.test", vol.Tags["csi.driver"])
}

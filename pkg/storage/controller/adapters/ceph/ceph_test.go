package ceph_test

import (
	"context"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/storage/controller"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/storage/controller/adapters/ceph"
	"github.com/stretchr/testify/require"
)

func TestCephControllerLifecycle(t *testing.T) {
	client := ceph.NewMemoryRBDClient("ceph-pool")
	c, err := ceph.New(ceph.Config{Pool: "ceph-pool", Client: client})
	require.NoError(t, err)

	ctx := context.Background()
	id, err := c.CreateVolume(ctx, controller.VolumeSpec{
		Name:   "data",
		SizeGB: 10,
		Type:   "ssd",
		Tags:   map[string]string{"type": "ssd", "env": "test"},
	})
	require.NoError(t, err)
	require.Contains(t, id, "rbd-")

	vol, err := c.GetVolume(ctx, id)
	require.NoError(t, err)
	require.Equal(t, controller.VolumeStatusAvailable, vol.Status)
	require.Equal(t, "ceph-pool", vol.Zone)
	require.Equal(t, "ceph-pool", vol.Tags["ceph.pool"])
	require.Equal(t, "ssd", vol.Type)

	require.NoError(t, c.AttachVolume(ctx, id, "node-1"))
	vol, err = c.GetVolume(ctx, id)
	require.NoError(t, err)
	require.Equal(t, controller.VolumeStatusAttached, vol.Status)
	require.Equal(t, "node-1", vol.AttachedTo)
	require.Contains(t, vol.Tags["ceph.device"], "ceph-pool")

	require.NoError(t, c.AttachVolume(ctx, id, "node-1")) // idempotent
	require.ErrorIs(t, c.AttachVolume(ctx, id, "node-2"), controller.ErrVolumeAttached)
	require.ErrorIs(t, c.DeleteVolume(ctx, id), controller.ErrVolumeAttached)

	require.NoError(t, c.DetachVolume(ctx, id))
	require.NoError(t, c.ResizeVolume(ctx, id, 20))
	vol, err = c.GetVolume(ctx, id)
	require.NoError(t, err)
	require.Equal(t, 20, vol.SizeGB)
	require.Equal(t, controller.VolumeStatusAvailable, vol.Status)

	require.ErrorIs(t, c.ResizeVolume(ctx, id, 5), controller.ErrInvalidSize)
	require.NoError(t, c.DeleteVolume(ctx, id))
	_, err = c.GetVolume(ctx, id)
	require.ErrorIs(t, err, controller.ErrVolumeNotFound)
}

func TestCephNewRequiresClient(t *testing.T) {
	_, err := ceph.New(ceph.Config{})
	require.Error(t, err)
}

func TestCephInvalidSize(t *testing.T) {
	c, err := ceph.New(ceph.Config{Client: ceph.NewMemoryRBDClient("")})
	require.NoError(t, err)
	_, err = c.CreateVolume(context.Background(), controller.VolumeSpec{Name: "x", SizeGB: 0})
	require.ErrorIs(t, err, controller.ErrInvalidSize)
}

func TestCephCreateFromSnapshotTag(t *testing.T) {
	c, err := ceph.New(ceph.Config{Client: ceph.NewMemoryRBDClient("rbd")})
	require.NoError(t, err)
	id, err := c.CreateVolume(context.Background(), controller.VolumeSpec{
		Name: "from-snap", SizeGB: 5, SnapshotID: "snap-1",
	})
	require.NoError(t, err)
	vol, err := c.GetVolume(context.Background(), id)
	require.NoError(t, err)
	require.Equal(t, "snap-1", vol.Tags["ceph.source_snapshot"])
}

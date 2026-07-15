package local

import (
	"context"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/storage/block"
	"github.com/stretchr/testify/require"
)

func TestLocalVolumeStore(t *testing.T) {
	dir := t.TempDir()
	store, err := New(dir)
	require.NoError(t, err)
	ctx := context.Background()

	vol, err := store.CreateVolume(ctx, block.CreateVolumeOptions{
		Name:   "v1",
		SizeGB: 10,
		Tags:   map[string]string{"env": "test"},
	})
	require.NoError(t, err)
	require.NotEmpty(t, vol.ID)

	got, err := store.GetVolume(ctx, vol.ID)
	require.NoError(t, err)
	require.Equal(t, "v1", got.Name)

	list, err := store.ListVolumes(ctx, block.ListOptions{})
	require.NoError(t, err)
	require.Len(t, list.Volumes, 1)

	require.NoError(t, store.AttachVolume(ctx, block.AttachVolumeOptions{
		VolumeID:   vol.ID,
		InstanceID: "i-1",
		Device:     "/dev/sdf",
	}))
	require.Error(t, store.DeleteVolume(ctx, vol.ID))
	require.NoError(t, store.DetachVolume(ctx, vol.ID, "i-1"))

	snap, err := store.CreateSnapshot(ctx, block.CreateSnapshotOptions{
		VolumeID:    vol.ID,
		Description: "snap",
	})
	require.NoError(t, err)
	_, err = store.GetSnapshot(ctx, snap.ID)
	require.NoError(t, err)
	require.NoError(t, store.DeleteSnapshot(ctx, snap.ID))

	resized, err := store.ResizeVolume(ctx, vol.ID, block.ResizeVolumeOptions{NewSizeGB: 20})
	require.NoError(t, err)
	require.Equal(t, int64(20), resized.SizeGB)

	require.NoError(t, store.DeleteVolume(ctx, vol.ID))
	_, err = store.GetVolume(ctx, vol.ID)
	require.Error(t, err)
}

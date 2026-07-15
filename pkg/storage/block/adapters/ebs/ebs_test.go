package ebs_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/storage/block"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/storage/block/adapters/ebs"
	"github.com/stretchr/testify/require"
)

func TestEBSStub_VolumeLifecycle(t *testing.T) {
	root := t.TempDir()
	store, err := ebs.New(ebs.Config{Root: root, Region: "us-west-2", AvailabilityZone: "us-west-2a"})
	require.NoError(t, err)

	ctx := context.Background()
	vol, err := store.CreateVolume(ctx, block.CreateVolumeOptions{
		Name:   "data",
		SizeGB: 100,
	})
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(vol.ID, "vol-"))
	require.Equal(t, "us-west-2a", vol.AvailabilityZone)
	require.Equal(t, "us-west-2", vol.Tags["ebs.amazonaws.com/region"])

	got, err := store.GetVolume(ctx, vol.ID)
	require.NoError(t, err)
	require.Equal(t, int64(100), got.SizeGB)

	require.NoError(t, store.AttachVolume(ctx, block.AttachVolumeOptions{
		VolumeID: vol.ID, InstanceID: "i-abc",
	}))
	require.ErrorIs(t, store.DeleteVolume(ctx, vol.ID), block.ErrVolumeInUse)
	require.NoError(t, store.DetachVolume(ctx, vol.ID, "i-abc"))

	snap, err := store.CreateSnapshot(ctx, block.CreateSnapshotOptions{VolumeID: vol.ID, Description: "backup"})
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(snap.ID, "snap-"))

	resized, err := store.ResizeVolume(ctx, vol.ID, block.ResizeVolumeOptions{NewSizeGB: 200})
	require.NoError(t, err)
	require.Equal(t, int64(200), resized.SizeGB)

	list, err := store.ListVolumes(ctx, block.ListOptions{})
	require.NoError(t, err)
	require.Len(t, list.Volumes, 1)

	require.NoError(t, store.DeleteSnapshot(ctx, snap.ID))
	require.NoError(t, store.DeleteVolume(ctx, vol.ID))
	_, err = os.Stat(filepath.Join(root, "volumes", vol.ID+".json"))
	require.True(t, os.IsNotExist(err))
}

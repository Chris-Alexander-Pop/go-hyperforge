package ebs_test

import (
	"context"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/storage/block"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/storage/block/adapters/ebs"
	"github.com/stretchr/testify/require"
)

func TestSDKStoreCreateAttachDetach(t *testing.T) {
	api := ebs.NewMemoryEC2API()
	store, err := ebs.NewSDKFromAPI(api, ebs.SDKConfig{Region: "us-west-2", AvailabilityZone: "us-west-2a"})
	require.NoError(t, err)

	ctx := context.Background()
	vol, err := store.CreateVolume(ctx, block.CreateVolumeOptions{
		Name:   "data",
		SizeGB: 20,
		Tags:   map[string]string{"env": "test"},
	})
	require.NoError(t, err)
	require.NotEmpty(t, vol.ID)
	require.Equal(t, int64(20), vol.SizeGB)
	require.Equal(t, "data", vol.Name)

	got, err := store.GetVolume(ctx, vol.ID)
	require.NoError(t, err)
	require.Equal(t, vol.ID, got.ID)

	require.NoError(t, store.AttachVolume(ctx, block.AttachVolumeOptions{
		VolumeID:   vol.ID,
		InstanceID: "i-abc",
		Device:     "/dev/sdf",
	}))
	got, err = store.GetVolume(ctx, vol.ID)
	require.NoError(t, err)
	require.Equal(t, block.VolumeStateInUse, got.State)
	require.Len(t, got.Attachments, 1)

	require.NoError(t, store.DetachVolume(ctx, vol.ID, "i-abc"))
	got, err = store.GetVolume(ctx, vol.ID)
	require.NoError(t, err)
	require.Equal(t, block.VolumeStateAvailable, got.State)

	snap, err := store.CreateSnapshot(ctx, block.CreateSnapshotOptions{
		VolumeID:    vol.ID,
		Description: "backup",
	})
	require.NoError(t, err)
	require.NotEmpty(t, snap.ID)

	list, err := store.ListVolumes(ctx, block.ListOptions{})
	require.NoError(t, err)
	require.Len(t, list.Volumes, 1)

	require.NoError(t, store.DeleteSnapshot(ctx, snap.ID))
	require.NoError(t, store.DeleteVolume(ctx, vol.ID))
}

func TestSDKFromAPIRequiresClient(t *testing.T) {
	_, err := ebs.NewSDKFromAPI(nil, ebs.SDKConfig{})
	require.Error(t, err)
}

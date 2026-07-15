package ebs_test

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
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

func TestSDKWaitUntilVolumeAvailable(t *testing.T) {
	api := ebs.NewMemoryEC2API()
	store, err := ebs.NewSDKFromAPI(api, ebs.SDKConfig{
		Region: "us-east-1", PollInterval: 5 * time.Millisecond,
	})
	require.NoError(t, err)

	ctx := context.Background()
	vol, err := store.CreateVolume(ctx, block.CreateVolumeOptions{SizeGB: 10})
	require.NoError(t, err)
	require.NoError(t, api.SetVolumeState(vol.ID, types.VolumeStateCreating))

	done := make(chan error, 1)
	go func() {
		done <- store.WaitUntilVolumeAvailable(ctx, vol.ID)
	}()

	time.Sleep(15 * time.Millisecond)
	require.NoError(t, api.SetVolumeState(vol.ID, types.VolumeStateAvailable))

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("WaitUntilVolumeAvailable timed out")
	}
}

func TestSDKWaitUntilVolumeInUse(t *testing.T) {
	api := ebs.NewMemoryEC2API()
	store, err := ebs.NewSDKFromAPI(api, ebs.SDKConfig{
		Region: "us-east-1", PollInterval: 5 * time.Millisecond,
	})
	require.NoError(t, err)

	ctx := context.Background()
	vol, err := store.CreateVolume(ctx, block.CreateVolumeOptions{SizeGB: 5})
	require.NoError(t, err)

	done := make(chan error, 1)
	go func() {
		done <- store.WaitUntilVolumeInUse(ctx, vol.ID)
	}()

	time.Sleep(10 * time.Millisecond)
	require.NoError(t, store.AttachVolume(ctx, block.AttachVolumeOptions{
		VolumeID: vol.ID, InstanceID: "i-1", Device: "/dev/sdf",
	}))

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("WaitUntilVolumeInUse timed out")
	}
}

func TestSDKWaitUntilVolumeDeleted(t *testing.T) {
	api := ebs.NewMemoryEC2API()
	store, err := ebs.NewSDKFromAPI(api, ebs.SDKConfig{
		Region: "us-east-1", PollInterval: 5 * time.Millisecond,
	})
	require.NoError(t, err)

	ctx := context.Background()
	vol, err := store.CreateVolume(ctx, block.CreateVolumeOptions{SizeGB: 5})
	require.NoError(t, err)

	done := make(chan error, 1)
	go func() {
		done <- store.WaitUntilVolumeDeleted(ctx, vol.ID)
	}()

	time.Sleep(10 * time.Millisecond)
	require.NoError(t, store.DeleteVolume(ctx, vol.ID))

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("WaitUntilVolumeDeleted timed out")
	}

	// Already gone is success
	require.NoError(t, store.WaitUntilVolumeDeleted(ctx, vol.ID))
}

func TestSDKWaitUntilSnapshotCompleted(t *testing.T) {
	api := ebs.NewMemoryEC2API()
	store, err := ebs.NewSDKFromAPI(api, ebs.SDKConfig{
		Region: "us-east-1", PollInterval: 5 * time.Millisecond,
	})
	require.NoError(t, err)

	ctx := context.Background()
	vol, err := store.CreateVolume(ctx, block.CreateVolumeOptions{SizeGB: 5})
	require.NoError(t, err)
	snap, err := store.CreateSnapshot(ctx, block.CreateSnapshotOptions{VolumeID: vol.ID})
	require.NoError(t, err)
	require.NoError(t, api.SetSnapshotState(snap.ID, types.SnapshotStatePending))

	done := make(chan error, 1)
	go func() {
		done <- store.WaitUntilSnapshotCompleted(ctx, snap.ID)
	}()

	time.Sleep(10 * time.Millisecond)
	require.NoError(t, api.SetSnapshotState(snap.ID, types.SnapshotStateCompleted))

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("WaitUntilSnapshotCompleted timed out")
	}
}

func TestSDKWaitersRespectContextCancel(t *testing.T) {
	api := ebs.NewMemoryEC2API()
	store, err := ebs.NewSDKFromAPI(api, ebs.SDKConfig{
		Region: "us-east-1", PollInterval: 50 * time.Millisecond,
	})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	vol, err := store.CreateVolume(ctx, block.CreateVolumeOptions{SizeGB: 1})
	require.NoError(t, err)
	require.NoError(t, api.SetVolumeState(vol.ID, types.VolumeStateCreating))

	done := make(chan error, 1)
	go func() {
		done <- store.WaitUntilVolumeAvailable(ctx, vol.ID)
	}()
	cancel()

	select {
	case err := <-done:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(2 * time.Second):
		t.Fatal("waiter did not observe cancel")
	}
}

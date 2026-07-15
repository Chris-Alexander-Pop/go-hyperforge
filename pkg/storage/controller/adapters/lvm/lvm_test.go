package lvm_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/storage/controller"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/storage/controller/adapters/lvm"
	"github.com/stretchr/testify/require"
)

func TestLVMControllerLifecycle(t *testing.T) {
	root := t.TempDir()
	c, err := lvm.New(lvm.Config{Root: root, VolumeGroup: "testvg"})
	require.NoError(t, err)

	ctx := context.Background()
	id, err := c.CreateVolume(ctx, controller.VolumeSpec{Name: "data", SizeGB: 2, Type: "ssd"})
	require.NoError(t, err)
	require.FileExists(t, filepath.Join(root, "volumes", id+".img"))
	require.FileExists(t, filepath.Join(root, "meta", id+".json"))

	vol, err := c.GetVolume(ctx, id)
	require.NoError(t, err)
	require.Equal(t, controller.VolumeStatusAvailable, vol.Status)
	require.Equal(t, "testvg", vol.Tags["lvm.volume_group"])

	require.NoError(t, c.AttachVolume(ctx, id, "node-1"))
	vol, err = c.GetVolume(ctx, id)
	require.NoError(t, err)
	require.Equal(t, controller.VolumeStatusAttached, vol.Status)
	require.Equal(t, "node-1", vol.AttachedTo)
	require.Contains(t, vol.Tags["lvm.device"], "testvg")

	require.ErrorIs(t, c.DeleteVolume(ctx, id), controller.ErrVolumeAttached)
	require.NoError(t, c.DetachVolume(ctx, id))
	require.NoError(t, c.ResizeVolume(ctx, id, 4))
	vol, err = c.GetVolume(ctx, id)
	require.NoError(t, err)
	require.Equal(t, 4, vol.SizeGB)

	require.NoError(t, c.DeleteVolume(ctx, id))
	_, err = os.Stat(filepath.Join(root, "meta", id+".json"))
	require.True(t, os.IsNotExist(err))
}

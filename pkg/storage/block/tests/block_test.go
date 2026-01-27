package tests

import (
	"context"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/storage/block"
	"github.com/chris-alexander-pop/system-design-library/pkg/storage/block/adapters/memory"
	"github.com/stretchr/testify/suite"
)

// VolumeStoreSuite provides a generic test suite for VolumeStore implementations.
type VolumeStoreSuite struct {
	suite.Suite
	store block.VolumeStore
	ctx   context.Context
}

// SetupTest runs before each test.
func (s *VolumeStoreSuite) SetupTest() {
	s.store = memory.New()
	s.ctx = context.Background()
}

func (s *VolumeStoreSuite) TestCreateVolume() {
	vol, err := s.store.CreateVolume(s.ctx, block.CreateVolumeOptions{
		Name:       "test-volume",
		SizeGB:     100,
		VolumeType: block.VolumeTypeSSD,
		Tags:       map[string]string{"env": "test"},
	})
	s.Require().NoError(err)
	s.NotEmpty(vol.ID)
	s.Equal("test-volume", vol.Name)
	s.Equal(int64(100), vol.SizeGB)
	s.Equal(block.VolumeTypeSSD, vol.VolumeType)
	s.Equal(block.VolumeStateAvailable, vol.State)
}

func (s *VolumeStoreSuite) TestCreateVolumeInvalidSize() {
	_, err := s.store.CreateVolume(s.ctx, block.CreateVolumeOptions{
		Name:   "invalid",
		SizeGB: 0,
	})
	s.Error(err)
}

func (s *VolumeStoreSuite) TestGetVolume() {
	vol, err := s.store.CreateVolume(s.ctx, block.CreateVolumeOptions{
		Name:   "get-test",
		SizeGB: 50,
	})
	s.Require().NoError(err)

	got, err := s.store.GetVolume(s.ctx, vol.ID)
	s.Require().NoError(err)
	s.Equal(vol.ID, got.ID)
	s.Equal(vol.Name, got.Name)
}

func (s *VolumeStoreSuite) TestGetVolumeNotFound() {
	_, err := s.store.GetVolume(s.ctx, "nonexistent")
	s.Error(err)
}

func (s *VolumeStoreSuite) TestListVolumes() {
	// Create some volumes
	for i := 0; i < 5; i++ {
		_, err := s.store.CreateVolume(s.ctx, block.CreateVolumeOptions{
			Name:   "list-vol",
			SizeGB: 10,
		})
		s.Require().NoError(err)
	}

	result, err := s.store.ListVolumes(s.ctx, block.ListOptions{})
	s.Require().NoError(err)
	s.Len(result.Volumes, 5)
}

func (s *VolumeStoreSuite) TestListVolumesWithLimit() {
	for i := 0; i < 10; i++ {
		_, err := s.store.CreateVolume(s.ctx, block.CreateVolumeOptions{
			Name:   "limit-vol",
			SizeGB: 10,
		})
		s.Require().NoError(err)
	}

	result, err := s.store.ListVolumes(s.ctx, block.ListOptions{Limit: 3})
	s.Require().NoError(err)
	s.Len(result.Volumes, 3)
}

func (s *VolumeStoreSuite) TestDeleteVolume() {
	vol, err := s.store.CreateVolume(s.ctx, block.CreateVolumeOptions{
		Name:   "delete-me",
		SizeGB: 20,
	})
	s.Require().NoError(err)

	err = s.store.DeleteVolume(s.ctx, vol.ID)
	s.Require().NoError(err)

	_, err = s.store.GetVolume(s.ctx, vol.ID)
	s.Error(err)
}

func (s *VolumeStoreSuite) TestDeleteVolumeNotFound() {
	err := s.store.DeleteVolume(s.ctx, "nonexistent")
	s.Error(err)
}

func (s *VolumeStoreSuite) TestDeleteVolumeInUse() {
	vol, err := s.store.CreateVolume(s.ctx, block.CreateVolumeOptions{
		Name:   "attached-vol",
		SizeGB: 30,
	})
	s.Require().NoError(err)

	err = s.store.AttachVolume(s.ctx, block.AttachVolumeOptions{
		VolumeID:   vol.ID,
		InstanceID: "i-12345",
		Device:     "/dev/sdf",
	})
	s.Require().NoError(err)

	err = s.store.DeleteVolume(s.ctx, vol.ID)
	s.Error(err) // Should fail because volume is attached
}

func (s *VolumeStoreSuite) TestResizeVolume() {
	vol, err := s.store.CreateVolume(s.ctx, block.CreateVolumeOptions{
		Name:   "resize-me",
		SizeGB: 50,
	})
	s.Require().NoError(err)

	resized, err := s.store.ResizeVolume(s.ctx, vol.ID, block.ResizeVolumeOptions{
		NewSizeGB:     100,
		NewVolumeType: block.VolumeTypeIOPS,
	})
	s.Require().NoError(err)
	s.Equal(int64(100), resized.SizeGB)
	s.Equal(block.VolumeTypeIOPS, resized.VolumeType)
}

func (s *VolumeStoreSuite) TestResizeVolumeShrinkFails() {
	vol, err := s.store.CreateVolume(s.ctx, block.CreateVolumeOptions{
		Name:   "no-shrink",
		SizeGB: 100,
	})
	s.Require().NoError(err)

	_, err = s.store.ResizeVolume(s.ctx, vol.ID, block.ResizeVolumeOptions{
		NewSizeGB: 50, // Smaller than current
	})
	s.Error(err)
}

func (s *VolumeStoreSuite) TestAttachDetachVolume() {
	vol, err := s.store.CreateVolume(s.ctx, block.CreateVolumeOptions{
		Name:   "attach-test",
		SizeGB: 25,
	})
	s.Require().NoError(err)
	s.Equal(block.VolumeStateAvailable, vol.State)

	// Attach
	err = s.store.AttachVolume(s.ctx, block.AttachVolumeOptions{
		VolumeID:   vol.ID,
		InstanceID: "i-abcdef",
		Device:     "/dev/xvdf",
	})
	s.Require().NoError(err)

	// Verify state changed
	vol, err = s.store.GetVolume(s.ctx, vol.ID)
	s.Require().NoError(err)
	s.Equal(block.VolumeStateInUse, vol.State)
	s.Len(vol.Attachments, 1)
	s.Equal("i-abcdef", vol.Attachments[0].InstanceID)

	// Detach
	err = s.store.DetachVolume(s.ctx, vol.ID, "i-abcdef")
	s.Require().NoError(err)

	vol, err = s.store.GetVolume(s.ctx, vol.ID)
	s.Require().NoError(err)
	s.Equal(block.VolumeStateAvailable, vol.State)
	s.Empty(vol.Attachments)
}

func (s *VolumeStoreSuite) TestDetachNotAttached() {
	vol, err := s.store.CreateVolume(s.ctx, block.CreateVolumeOptions{
		Name:   "not-attached",
		SizeGB: 10,
	})
	s.Require().NoError(err)

	err = s.store.DetachVolume(s.ctx, vol.ID, "i-unknown")
	s.Error(err)
}

func (s *VolumeStoreSuite) TestSnapshotLifecycle() {
	// Create volume
	vol, err := s.store.CreateVolume(s.ctx, block.CreateVolumeOptions{
		Name:   "snapshot-source",
		SizeGB: 50,
	})
	s.Require().NoError(err)

	// Create snapshot
	snap, err := s.store.CreateSnapshot(s.ctx, block.CreateSnapshotOptions{
		VolumeID:    vol.ID,
		Description: "test snapshot",
		Tags:        map[string]string{"backup": "daily"},
	})
	s.Require().NoError(err)
	s.NotEmpty(snap.ID)
	s.Equal(vol.ID, snap.VolumeID)
	s.Equal(vol.SizeGB, snap.SizeGB)
	s.Equal("test snapshot", snap.Description)

	// Get snapshot
	got, err := s.store.GetSnapshot(s.ctx, snap.ID)
	s.Require().NoError(err)
	s.Equal(snap.ID, got.ID)

	// Delete snapshot
	err = s.store.DeleteSnapshot(s.ctx, snap.ID)
	s.Require().NoError(err)

	_, err = s.store.GetSnapshot(s.ctx, snap.ID)
	s.Error(err)
}

func (s *VolumeStoreSuite) TestSnapshotOfNonexistentVolume() {
	_, err := s.store.CreateSnapshot(s.ctx, block.CreateSnapshotOptions{
		VolumeID: "nonexistent",
	})
	s.Error(err)
}

// TestVolumeStoreSuite runs the test suite.
func TestVolumeStoreSuite(t *testing.T) {
	suite.Run(t, new(VolumeStoreSuite))
}

package tests

import (
	"context"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/compute/vm"
	"github.com/chris-alexander-pop/system-design-library/pkg/compute/vm/adapters/memory"
	"github.com/stretchr/testify/suite"
)

// VMManagerSuite provides a generic test suite for VMManager implementations.
type VMManagerSuite struct {
	suite.Suite
	manager vm.VMManager
	ctx     context.Context
}

// SetupTest runs before each test.
func (s *VMManagerSuite) SetupTest() {
	s.manager = memory.New()
	s.ctx = context.Background()
}

func (s *VMManagerSuite) TestCreateAndGetInstance() {
	instance, err := s.manager.Create(s.ctx, vm.CreateOptions{
		Name:         "test-vm",
		InstanceType: "t3.small",
		ImageID:      "ami-12345",
		Tags:         map[string]string{"env": "test"},
	})
	s.Require().NoError(err)
	s.NotEmpty(instance.ID)
	s.Equal("test-vm", instance.Name)
	s.Equal(vm.InstanceStateRunning, instance.State)
	s.NotEmpty(instance.PublicIP)
	s.NotEmpty(instance.PrivateIP)

	got, err := s.manager.Get(s.ctx, instance.ID)
	s.Require().NoError(err)
	s.Equal(instance.ID, got.ID)
}

func (s *VMManagerSuite) TestGetNotFound() {
	_, err := s.manager.Get(s.ctx, "nonexistent")
	s.Error(err)
}

func (s *VMManagerSuite) TestListInstances() {
	for i := 0; i < 3; i++ {
		_, err := s.manager.Create(s.ctx, vm.CreateOptions{
			Name:    "vm-" + string(rune('0'+i)),
			ImageID: "ami-12345",
		})
		s.Require().NoError(err)
	}

	result, err := s.manager.List(s.ctx, vm.ListOptions{})
	s.Require().NoError(err)
	s.Len(result.Instances, 3)
}

func (s *VMManagerSuite) TestListWithStateFilter() {
	// Create running instance
	running, err := s.manager.Create(s.ctx, vm.CreateOptions{Name: "running", ImageID: "ami-123"})
	s.Require().NoError(err)

	// Create and stop another instance
	stopped, err := s.manager.Create(s.ctx, vm.CreateOptions{Name: "stopped", ImageID: "ami-123"})
	s.Require().NoError(err)
	err = s.manager.Stop(s.ctx, stopped.ID)
	s.Require().NoError(err)

	// List only running
	result, err := s.manager.List(s.ctx, vm.ListOptions{State: vm.InstanceStateRunning})
	s.Require().NoError(err)
	s.Len(result.Instances, 1)
	s.Equal(running.ID, result.Instances[0].ID)
}

func (s *VMManagerSuite) TestStopAndStartInstance() {
	instance, err := s.manager.Create(s.ctx, vm.CreateOptions{Name: "lifecycle", ImageID: "ami-123"})
	s.Require().NoError(err)
	s.Equal(vm.InstanceStateRunning, instance.State)

	// Stop
	err = s.manager.Stop(s.ctx, instance.ID)
	s.Require().NoError(err)

	instance, err = s.manager.Get(s.ctx, instance.ID)
	s.Require().NoError(err)
	s.Equal(vm.InstanceStateStopped, instance.State)

	// Start
	err = s.manager.Start(s.ctx, instance.ID)
	s.Require().NoError(err)

	instance, err = s.manager.Get(s.ctx, instance.ID)
	s.Require().NoError(err)
	s.Equal(vm.InstanceStateRunning, instance.State)
}

func (s *VMManagerSuite) TestStopNotRunning() {
	instance, err := s.manager.Create(s.ctx, vm.CreateOptions{Name: "test", ImageID: "ami-123"})
	s.Require().NoError(err)

	err = s.manager.Stop(s.ctx, instance.ID)
	s.Require().NoError(err)

	// Try to stop again
	err = s.manager.Stop(s.ctx, instance.ID)
	s.Error(err)
}

func (s *VMManagerSuite) TestStartNotStopped() {
	instance, err := s.manager.Create(s.ctx, vm.CreateOptions{Name: "test", ImageID: "ami-123"})
	s.Require().NoError(err)

	// Try to start already running
	err = s.manager.Start(s.ctx, instance.ID)
	s.Error(err)
}

func (s *VMManagerSuite) TestReboot() {
	instance, err := s.manager.Create(s.ctx, vm.CreateOptions{Name: "reboot-test", ImageID: "ami-123"})
	s.Require().NoError(err)

	err = s.manager.Reboot(s.ctx, instance.ID)
	s.NoError(err)

	instance, err = s.manager.Get(s.ctx, instance.ID)
	s.Require().NoError(err)
	s.Equal(vm.InstanceStateRunning, instance.State)
}

func (s *VMManagerSuite) TestTerminate() {
	instance, err := s.manager.Create(s.ctx, vm.CreateOptions{Name: "terminate-me", ImageID: "ami-123"})
	s.Require().NoError(err)

	err = s.manager.Terminate(s.ctx, instance.ID)
	s.Require().NoError(err)

	instance, err = s.manager.Get(s.ctx, instance.ID)
	s.Require().NoError(err)
	s.Equal(vm.InstanceStateTerminated, instance.State)
}

func (s *VMManagerSuite) TestUpdateTags() {
	instance, err := s.manager.Create(s.ctx, vm.CreateOptions{
		Name:    "tag-test",
		ImageID: "ami-123",
		Tags:    map[string]string{"env": "test"},
	})
	s.Require().NoError(err)

	err = s.manager.UpdateTags(s.ctx, instance.ID, map[string]string{"app": "web", "version": "1.0"})
	s.Require().NoError(err)

	instance, err = s.manager.Get(s.ctx, instance.ID)
	s.Require().NoError(err)
	s.Equal("test", instance.Tags["env"])
	s.Equal("web", instance.Tags["app"])
	s.Equal("1.0", instance.Tags["version"])
}

func (s *VMManagerSuite) TestGetConsoleOutput() {
	instance, err := s.manager.Create(s.ctx, vm.CreateOptions{Name: "console-test", ImageID: "ami-123"})
	s.Require().NoError(err)

	output, err := s.manager.GetConsoleOutput(s.ctx, instance.ID)
	s.Require().NoError(err)
	s.NotEmpty(output)
	s.Contains(output, instance.ID)
}

// TestVMManagerSuite runs the test suite.
func TestVMManagerSuite(t *testing.T) {
	suite.Run(t, new(VMManagerSuite))
}

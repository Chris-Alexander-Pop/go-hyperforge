package tests

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/compute/container"
	"github.com/chris-alexander-pop/system-design-library/pkg/compute/container/adapters/memory"
	"github.com/stretchr/testify/suite"
)

// ContainerRuntimeSuite provides a generic test suite for ContainerRuntime implementations.
type ContainerRuntimeSuite struct {
	suite.Suite
	runtime container.ContainerRuntime
	ctx     context.Context
}

// SetupTest runs before each test.
func (s *ContainerRuntimeSuite) SetupTest() {
	s.runtime = memory.New()
	s.ctx = context.Background()
}

func (s *ContainerRuntimeSuite) TestCreateAndGetContainer() {
	ctr, err := s.runtime.Create(s.ctx, container.CreateOptions{
		Name:   "test-container",
		Image:  "nginx:latest",
		Labels: map[string]string{"env": "test"},
	})
	s.Require().NoError(err)
	s.NotEmpty(ctr.ID)
	s.Equal("test-container", ctr.Name)
	s.Equal("nginx:latest", ctr.Image)
	s.Equal(container.ContainerStateCreated, ctr.State)

	got, err := s.runtime.Get(s.ctx, ctr.ID)
	s.Require().NoError(err)
	s.Equal(ctr.ID, got.ID)
}

func (s *ContainerRuntimeSuite) TestCreateWithNameConflict() {
	_, err := s.runtime.Create(s.ctx, container.CreateOptions{Name: "duplicate", Image: "nginx"})
	s.Require().NoError(err)

	_, err = s.runtime.Create(s.ctx, container.CreateOptions{Name: "duplicate", Image: "nginx"})
	s.Error(err)
}

func (s *ContainerRuntimeSuite) TestGetNotFound() {
	_, err := s.runtime.Get(s.ctx, "nonexistent")
	s.Error(err)
}

func (s *ContainerRuntimeSuite) TestListContainers() {
	for i := 0; i < 3; i++ {
		ctr, err := s.runtime.Create(s.ctx, container.CreateOptions{Image: "nginx"})
		s.Require().NoError(err)
		err = s.runtime.Start(s.ctx, ctr.ID)
		s.Require().NoError(err)
	}

	containers, err := s.runtime.List(s.ctx, container.ListOptions{})
	s.Require().NoError(err)
	s.Len(containers, 3)
}

func (s *ContainerRuntimeSuite) TestStartAndStopContainer() {
	ctr, err := s.runtime.Create(s.ctx, container.CreateOptions{Image: "nginx"})
	s.Require().NoError(err)

	err = s.runtime.Start(s.ctx, ctr.ID)
	s.Require().NoError(err)

	ctr, err = s.runtime.Get(s.ctx, ctr.ID)
	s.Require().NoError(err)
	s.Equal(container.ContainerStateRunning, ctr.State)

	err = s.runtime.Stop(s.ctx, ctr.ID, 10*time.Second)
	s.Require().NoError(err)

	ctr, err = s.runtime.Get(s.ctx, ctr.ID)
	s.Require().NoError(err)
	s.Equal(container.ContainerStateExited, ctr.State)
}

func (s *ContainerRuntimeSuite) TestStartAlreadyRunning() {
	ctr, err := s.runtime.Create(s.ctx, container.CreateOptions{Image: "nginx"})
	s.Require().NoError(err)

	err = s.runtime.Start(s.ctx, ctr.ID)
	s.Require().NoError(err)

	err = s.runtime.Start(s.ctx, ctr.ID)
	s.Error(err)
}

func (s *ContainerRuntimeSuite) TestStopNotRunning() {
	ctr, err := s.runtime.Create(s.ctx, container.CreateOptions{Image: "nginx"})
	s.Require().NoError(err)

	err = s.runtime.Stop(s.ctx, ctr.ID, 10*time.Second)
	s.Error(err)
}

func (s *ContainerRuntimeSuite) TestKill() {
	ctr, err := s.runtime.Create(s.ctx, container.CreateOptions{Image: "nginx"})
	s.Require().NoError(err)

	err = s.runtime.Start(s.ctx, ctr.ID)
	s.Require().NoError(err)

	err = s.runtime.Kill(s.ctx, ctr.ID, "SIGKILL")
	s.Require().NoError(err)

	ctr, err = s.runtime.Get(s.ctx, ctr.ID)
	s.Require().NoError(err)
	s.Equal(container.ContainerStateExited, ctr.State)
	s.Equal(137, ctr.ExitCode)
}

func (s *ContainerRuntimeSuite) TestRemove() {
	ctr, err := s.runtime.Create(s.ctx, container.CreateOptions{Image: "nginx"})
	s.Require().NoError(err)

	err = s.runtime.Remove(s.ctx, ctr.ID, false)
	s.Require().NoError(err)

	_, err = s.runtime.Get(s.ctx, ctr.ID)
	s.Error(err)
}

func (s *ContainerRuntimeSuite) TestRemoveRunningFails() {
	ctr, err := s.runtime.Create(s.ctx, container.CreateOptions{Image: "nginx"})
	s.Require().NoError(err)

	err = s.runtime.Start(s.ctx, ctr.ID)
	s.Require().NoError(err)

	err = s.runtime.Remove(s.ctx, ctr.ID, false)
	s.Error(err)

	// Force remove should work
	err = s.runtime.Remove(s.ctx, ctr.ID, true)
	s.NoError(err)
}

func (s *ContainerRuntimeSuite) TestLogs() {
	ctr, err := s.runtime.Create(s.ctx, container.CreateOptions{Image: "nginx"})
	s.Require().NoError(err)

	logs, err := s.runtime.Logs(s.ctx, ctr.ID, false)
	s.Require().NoError(err)
	defer logs.Close()

	data, err := io.ReadAll(logs)
	s.Require().NoError(err)
	s.Contains(string(data), "created")
}

func (s *ContainerRuntimeSuite) TestExec() {
	ctr, err := s.runtime.Create(s.ctx, container.CreateOptions{Image: "nginx"})
	s.Require().NoError(err)

	err = s.runtime.Start(s.ctx, ctr.ID)
	s.Require().NoError(err)

	result, err := s.runtime.Exec(s.ctx, ctr.ID, container.ExecOptions{
		Command: []string{"echo", "hello"},
	})
	s.Require().NoError(err)
	s.Equal(0, result.ExitCode)
	s.Contains(result.Stdout, "echo")
}

func (s *ContainerRuntimeSuite) TestExecNotRunning() {
	ctr, err := s.runtime.Create(s.ctx, container.CreateOptions{Image: "nginx"})
	s.Require().NoError(err)

	_, err = s.runtime.Exec(s.ctx, ctr.ID, container.ExecOptions{Command: []string{"echo"}})
	s.Error(err)
}

func (s *ContainerRuntimeSuite) TestStats() {
	ctr, err := s.runtime.Create(s.ctx, container.CreateOptions{Image: "nginx"})
	s.Require().NoError(err)

	err = s.runtime.Start(s.ctx, ctr.ID)
	s.Require().NoError(err)

	stats, err := s.runtime.Stats(s.ctx, ctr.ID)
	s.Require().NoError(err)
	s.True(stats.CPUPercent > 0)
	s.True(stats.MemoryUsage > 0)
}

// TestContainerRuntimeSuite runs the test suite.
func TestContainerRuntimeSuite(t *testing.T) {
	suite.Run(t, new(ContainerRuntimeSuite))
}

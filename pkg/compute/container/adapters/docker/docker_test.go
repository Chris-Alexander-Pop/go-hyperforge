package docker

import (
	"bufio"
	"context"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/compute/container"
	"github.com/docker/docker/api/types"
	dockerccontainer "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

type mockDocker struct {
	created   map[string]*dockerccontainer.InspectResponse
	createErr error
	startErr  error
	stopErr   error
	removeErr error
	logs      string
	execOut   string
	execCode  int
	statsJSON string
}

func newMockDocker() *mockDocker {
	return &mockDocker{created: map[string]*dockerccontainer.InspectResponse{}}
}

func (m *mockDocker) ContainerCreate(ctx context.Context, config *dockerccontainer.Config, hostConfig *dockerccontainer.HostConfig, networkingConfig *network.NetworkingConfig, platform *ocispec.Platform, containerName string) (dockerccontainer.CreateResponse, error) {
	if m.createErr != nil {
		return dockerccontainer.CreateResponse{}, m.createErr
	}
	id := "cid-" + containerName
	if containerName == "" {
		id = "cid-anon"
	}
	insp := &dockerccontainer.InspectResponse{
		ContainerJSONBase: &dockerccontainer.ContainerJSONBase{
			ID:         id,
			Name:       "/" + containerName,
			Created:    time.Now().Format(time.RFC3339Nano),
			State:      &dockerccontainer.State{Status: dockerccontainer.StateCreated},
			HostConfig: hostConfig,
		},
		Config: config,
		NetworkSettings: &dockerccontainer.NetworkSettings{
			Networks: map[string]*network.EndpointSettings{
				"bridge": {IPAddress: "172.17.0.2"},
			},
		},
	}
	m.created[id] = insp
	return dockerccontainer.CreateResponse{ID: id}, nil
}

func (m *mockDocker) ContainerInspect(ctx context.Context, containerID string) (dockerccontainer.InspectResponse, error) {
	insp, ok := m.created[containerID]
	if !ok {
		return dockerccontainer.InspectResponse{}, errNotFound("no such container")
	}
	return *insp, nil
}

func (m *mockDocker) ContainerList(ctx context.Context, options dockerccontainer.ListOptions) ([]dockerccontainer.Summary, error) {
	out := make([]dockerccontainer.Summary, 0, len(m.created))
	for _, insp := range m.created {
		out = append(out, dockerccontainer.Summary{
			ID:      insp.ID,
			Names:   []string{insp.Name},
			Image:   insp.Config.Image,
			State:   insp.State.Status,
			Created: time.Now().Unix(),
			Labels:  insp.Config.Labels,
		})
	}
	return out, nil
}

func (m *mockDocker) ContainerStart(ctx context.Context, containerID string, options dockerccontainer.StartOptions) error {
	if m.startErr != nil {
		return m.startErr
	}
	insp, ok := m.created[containerID]
	if !ok {
		return errNotFound("missing")
	}
	insp.State.Status = dockerccontainer.StateRunning
	return nil
}

func (m *mockDocker) ContainerStop(ctx context.Context, containerID string, options dockerccontainer.StopOptions) error {
	if m.stopErr != nil {
		return m.stopErr
	}
	insp, ok := m.created[containerID]
	if !ok {
		return errNotFound("missing")
	}
	insp.State.Status = dockerccontainer.StateExited
	return nil
}

func (m *mockDocker) ContainerKill(ctx context.Context, containerID, signal string) error {
	return m.ContainerStop(ctx, containerID, dockerccontainer.StopOptions{})
}

func (m *mockDocker) ContainerRemove(ctx context.Context, containerID string, options dockerccontainer.RemoveOptions) error {
	if m.removeErr != nil {
		return m.removeErr
	}
	if _, ok := m.created[containerID]; !ok {
		return errNotFound("missing")
	}
	delete(m.created, containerID)
	return nil
}

func (m *mockDocker) ContainerLogs(ctx context.Context, containerID string, options dockerccontainer.LogsOptions) (io.ReadCloser, error) {
	if _, ok := m.created[containerID]; !ok {
		return nil, errNotFound("missing")
	}
	return io.NopCloser(strings.NewReader(m.logs)), nil
}

func (m *mockDocker) ContainerExecCreate(ctx context.Context, c string, options dockerccontainer.ExecOptions) (dockerccontainer.ExecCreateResponse, error) {
	if _, ok := m.created[c]; !ok {
		return dockerccontainer.ExecCreateResponse{}, errNotFound("missing")
	}
	return dockerccontainer.ExecCreateResponse{ID: "exec-1"}, nil
}

func (m *mockDocker) ContainerExecAttach(ctx context.Context, execID string, options dockerccontainer.ExecAttachOptions) (types.HijackedResponse, error) {
	c1, c2 := net.Pipe()
	go func() {
		_, _ = c2.Write([]byte(m.execOut))
		_ = c2.Close()
	}()
	return types.HijackedResponse{Conn: c1, Reader: bufio.NewReader(c1)}, nil
}

func (m *mockDocker) ContainerExecInspect(ctx context.Context, execID string) (dockerccontainer.ExecInspect, error) {
	return dockerccontainer.ExecInspect{ExitCode: m.execCode}, nil
}

func (m *mockDocker) ContainerWait(ctx context.Context, containerID string, condition dockerccontainer.WaitCondition) (<-chan dockerccontainer.WaitResponse, <-chan error) {
	st := make(chan dockerccontainer.WaitResponse, 1)
	errCh := make(chan error, 1)
	st <- dockerccontainer.WaitResponse{StatusCode: 0}
	return st, errCh
}

func (m *mockDocker) ContainerStatsOneShot(ctx context.Context, containerID string) (dockerccontainer.StatsResponseReader, error) {
	if _, ok := m.created[containerID]; !ok {
		return dockerccontainer.StatsResponseReader{}, errNotFound("missing")
	}
	body := m.statsJSON
	if body == "" {
		body = `{"cpu_stats":{"cpu_usage":{"total_usage":200,"percpu_usage":[100,100]},"system_usage":400,"online_cpus":2},"precpu_stats":{"cpu_usage":{"total_usage":100,"percpu_usage":[50,50]},"system_usage":200,"online_cpus":2},"memory_stats":{"usage":1048576,"limit":2097152},"networks":{"eth0":{"rx_bytes":10,"tx_bytes":20}}}`
	}
	return dockerccontainer.StatsResponseReader{Body: io.NopCloser(strings.NewReader(body))}, nil
}

func (m *mockDocker) Close() error { return nil }

type notFoundErr struct{ msg string }

func (e notFoundErr) Error() string { return e.msg }
func (e notFoundErr) NotFound()     {}

func errNotFound(msg string) error { return notFoundErr{msg: msg} }

func TestDockerCreateStartStopRemoveLogsExecStats(t *testing.T) {
	mock := newMockDocker()
	mock.logs = "hello"
	mock.execOut = "ok"
	rt := NewWithClient(mock, Config{})
	ctx := context.Background()

	c, err := rt.Create(ctx, container.CreateOptions{
		Name: "web", Image: "nginx:latest", Memory: 128, CPU: 0.5,
		Env:   map[string]string{"A": "1"},
		Ports: []container.PortMapping{{ContainerPort: 80, HostPort: 8080}},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if c.Image != "nginx:latest" || c.State != container.ContainerStateCreated {
		t.Fatalf("unexpected: %+v", c)
	}

	if err := rt.Start(ctx, c.ID); err != nil {
		t.Fatal(err)
	}
	got, err := rt.Get(ctx, c.ID)
	if err != nil || got.State != container.ContainerStateRunning {
		t.Fatalf("Get after start: %v %+v", err, got)
	}

	list, err := rt.List(ctx, container.ListOptions{All: true})
	if err != nil || len(list) != 1 {
		t.Fatalf("List: %v len=%d", err, len(list))
	}

	logs, err := rt.Logs(ctx, c.ID, false)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(logs)
	if string(b) != "hello" {
		t.Fatalf("logs: %q", b)
	}

	exec, err := rt.Exec(ctx, c.ID, container.ExecOptions{Command: []string{"echo", "hi"}})
	if err != nil || exec.Stdout != "ok" {
		t.Fatalf("Exec: %v %+v", err, exec)
	}

	stats, err := rt.Stats(ctx, c.ID)
	if err != nil || stats.MemoryUsage != 1048576 {
		t.Fatalf("Stats: %v %+v", err, stats)
	}

	code, err := rt.Wait(ctx, c.ID)
	if err != nil || code != 0 {
		t.Fatalf("Wait: %v %d", err, code)
	}

	if err := rt.Stop(ctx, c.ID, time.Second); err != nil {
		t.Fatal(err)
	}
	if err := rt.Remove(ctx, c.ID, true); err != nil {
		t.Fatal(err)
	}
}

func TestDockerCreateRequiresImage(t *testing.T) {
	rt := NewWithClient(newMockDocker(), Config{})
	_, err := rt.Create(context.Background(), container.CreateOptions{})
	if err == nil {
		t.Fatal("expected error")
	}
}

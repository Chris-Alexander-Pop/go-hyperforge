package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/compute/container"
	pkgerrors "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/docker/docker/api/types"
	dockerccontainer "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// Config holds Docker adapter configuration.
type Config struct {
	Host       string `env:"DOCKER_HOST" env-default:"unix:///var/run/docker.sock"`
	APIVersion string `env:"DOCKER_API_VERSION"`
}

// ContainerAPI is the Docker Engine subset used by Runtime.
type ContainerAPI interface {
	ContainerCreate(ctx context.Context, config *dockerccontainer.Config, hostConfig *dockerccontainer.HostConfig, networkingConfig *network.NetworkingConfig, platform *ocispec.Platform, containerName string) (dockerccontainer.CreateResponse, error)
	ContainerInspect(ctx context.Context, containerID string) (dockerccontainer.InspectResponse, error)
	ContainerList(ctx context.Context, options dockerccontainer.ListOptions) ([]dockerccontainer.Summary, error)
	ContainerStart(ctx context.Context, containerID string, options dockerccontainer.StartOptions) error
	ContainerStop(ctx context.Context, containerID string, options dockerccontainer.StopOptions) error
	ContainerKill(ctx context.Context, containerID, signal string) error
	ContainerRemove(ctx context.Context, containerID string, options dockerccontainer.RemoveOptions) error
	ContainerLogs(ctx context.Context, containerID string, options dockerccontainer.LogsOptions) (io.ReadCloser, error)
	ContainerExecCreate(ctx context.Context, container string, options dockerccontainer.ExecOptions) (dockerccontainer.ExecCreateResponse, error)
	ContainerExecAttach(ctx context.Context, execID string, options dockerccontainer.ExecAttachOptions) (types.HijackedResponse, error)
	ContainerExecInspect(ctx context.Context, execID string) (dockerccontainer.ExecInspect, error)
	ContainerWait(ctx context.Context, containerID string, condition dockerccontainer.WaitCondition) (<-chan dockerccontainer.WaitResponse, <-chan error)
	ContainerStatsOneShot(ctx context.Context, containerID string) (dockerccontainer.StatsResponseReader, error)
	Close() error
}

// Runtime implements container.ContainerRuntime for Docker.
type Runtime struct {
	client ContainerAPI
	config Config
}

// New creates a Docker runtime connected to the daemon.
func New(cfg Config) (*Runtime, error) {
	opts := []client.Opt{client.FromEnv, client.WithAPIVersionNegotiation()}
	if cfg.Host != "" {
		opts = append(opts, client.WithHost(cfg.Host))
	}
	if cfg.APIVersion != "" {
		opts = append(opts, client.WithVersion(cfg.APIVersion))
	}
	cli, err := client.NewClientWithOpts(opts...)
	if err != nil {
		return nil, pkgerrors.Internal("failed to create docker client", err)
	}
	return NewWithClient(cli, cfg), nil
}

// NewWithClient creates a Runtime with an injected ContainerAPI.
func NewWithClient(cli ContainerAPI, cfg Config) *Runtime {
	return &Runtime{client: cli, config: cfg}
}

// Close closes the underlying Docker client when applicable.
func (r *Runtime) Close() error {
	if r.client != nil {
		return r.client.Close()
	}
	return nil
}

func (r *Runtime) Create(ctx context.Context, opts container.CreateOptions) (*container.Container, error) {
	if opts.Image == "" {
		return nil, pkgerrors.InvalidArgument("image is required", nil)
	}

	env := make([]string, 0, len(opts.Env))
	for k, v := range opts.Env {
		env = append(env, k+"="+v)
	}

	exposed := nat.PortSet{}
	portBindings := nat.PortMap{}
	for _, p := range opts.Ports {
		proto := p.Protocol
		if proto == "" {
			proto = "tcp"
		}
		port, err := nat.NewPort(proto, fmt.Sprintf("%d", p.ContainerPort))
		if err != nil {
			return nil, pkgerrors.InvalidArgument("invalid port mapping", err)
		}
		exposed[port] = struct{}{}
		if p.HostPort > 0 {
			portBindings[port] = []nat.PortBinding{{HostPort: fmt.Sprintf("%d", p.HostPort)}}
		}
	}

	cfg := &dockerccontainer.Config{
		Image:        opts.Image,
		Cmd:          opts.Command,
		Entrypoint:   opts.Entrypoint,
		Env:          env,
		Labels:       opts.Labels,
		WorkingDir:   opts.WorkDir,
		User:         opts.User,
		ExposedPorts: exposed,
	}

	hostCfg := &dockerccontainer.HostConfig{
		PortBindings: portBindings,
		NetworkMode:  dockerccontainer.NetworkMode(opts.NetworkMode),
	}
	if opts.Memory > 0 {
		hostCfg.Resources.Memory = opts.Memory * 1024 * 1024
	}
	if opts.CPU > 0 {
		hostCfg.Resources.NanoCPUs = int64(opts.CPU * 1e9)
	}
	switch opts.RestartPolicy {
	case container.RestartPolicyAlways:
		hostCfg.RestartPolicy = dockerccontainer.RestartPolicy{Name: dockerccontainer.RestartPolicyAlways}
	case container.RestartPolicyOnFailure:
		hostCfg.RestartPolicy = dockerccontainer.RestartPolicy{Name: dockerccontainer.RestartPolicyOnFailure}
	case container.RestartPolicyUnlessStopped:
		hostCfg.RestartPolicy = dockerccontainer.RestartPolicy{Name: dockerccontainer.RestartPolicyUnlessStopped}
	default:
		hostCfg.RestartPolicy = dockerccontainer.RestartPolicy{Name: dockerccontainer.RestartPolicyDisabled}
	}
	for _, v := range opts.Volumes {
		hostCfg.Binds = append(hostCfg.Binds, fmt.Sprintf("%s:%s", v.Source, v.Target))
		if v.ReadOnly {
			hostCfg.Binds[len(hostCfg.Binds)-1] += ":ro"
		}
	}

	resp, err := r.client.ContainerCreate(ctx, cfg, hostCfg, nil, nil, opts.Name)
	if err != nil {
		if strings.Contains(err.Error(), "Conflict") {
			return nil, container.ErrNameConflict
		}
		return nil, pkgerrors.Internal("failed to create container", err)
	}
	return r.Get(ctx, resp.ID)
}

func (r *Runtime) Get(ctx context.Context, containerID string) (*container.Container, error) {
	insp, err := r.client.ContainerInspect(ctx, containerID)
	if err != nil {
		if client.IsErrNotFound(err) {
			return nil, container.ErrContainerNotFound
		}
		return nil, pkgerrors.Internal("failed to inspect container", err)
	}
	return mapInspect(insp), nil
}

func (r *Runtime) List(ctx context.Context, opts container.ListOptions) ([]*container.Container, error) {
	listOpts := dockerccontainer.ListOptions{All: opts.All}
	if opts.Limit > 0 {
		listOpts.Limit = opts.Limit
	}
	if len(opts.Labels) > 0 {
		f := filters.NewArgs()
		for k, v := range opts.Labels {
			f.Add("label", k+"="+v)
		}
		listOpts.Filters = f
	}
	summaries, err := r.client.ContainerList(ctx, listOpts)
	if err != nil {
		return nil, pkgerrors.Internal("failed to list containers", err)
	}
	out := make([]*container.Container, 0, len(summaries))
	for _, s := range summaries {
		name := ""
		if len(s.Names) > 0 {
			name = strings.TrimPrefix(s.Names[0], "/")
		}
		out = append(out, &container.Container{
			ID:        s.ID,
			Name:      name,
			Image:     s.Image,
			State:     mapDockerState(s.State),
			Labels:    s.Labels,
			CreatedAt: time.Unix(s.Created, 0),
		})
	}
	return out, nil
}

func (r *Runtime) Start(ctx context.Context, containerID string) error {
	err := r.client.ContainerStart(ctx, containerID, dockerccontainer.StartOptions{})
	if err != nil {
		if client.IsErrNotFound(err) {
			return container.ErrContainerNotFound
		}
		return pkgerrors.Internal("failed to start container", err)
	}
	return nil
}

func (r *Runtime) Stop(ctx context.Context, containerID string, timeout time.Duration) error {
	var secs *int
	if timeout > 0 {
		s := int(timeout.Seconds())
		secs = &s
	}
	err := r.client.ContainerStop(ctx, containerID, dockerccontainer.StopOptions{Timeout: secs})
	if err != nil {
		if client.IsErrNotFound(err) {
			return container.ErrContainerNotFound
		}
		return pkgerrors.Internal("failed to stop container", err)
	}
	return nil
}

func (r *Runtime) Kill(ctx context.Context, containerID string, signal string) error {
	if signal == "" {
		signal = "SIGKILL"
	}
	err := r.client.ContainerKill(ctx, containerID, signal)
	if err != nil {
		if client.IsErrNotFound(err) {
			return container.ErrContainerNotFound
		}
		return pkgerrors.Internal("failed to kill container", err)
	}
	return nil
}

func (r *Runtime) Remove(ctx context.Context, containerID string, force bool) error {
	err := r.client.ContainerRemove(ctx, containerID, dockerccontainer.RemoveOptions{Force: force})
	if err != nil {
		if client.IsErrNotFound(err) {
			return container.ErrContainerNotFound
		}
		return pkgerrors.Internal("failed to remove container", err)
	}
	return nil
}

func (r *Runtime) Logs(ctx context.Context, containerID string, follow bool) (io.ReadCloser, error) {
	rc, err := r.client.ContainerLogs(ctx, containerID, dockerccontainer.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     follow,
	})
	if err != nil {
		if client.IsErrNotFound(err) {
			return nil, container.ErrContainerNotFound
		}
		return nil, pkgerrors.Internal("failed to get container logs", err)
	}
	return rc, nil
}

func (r *Runtime) Exec(ctx context.Context, containerID string, opts container.ExecOptions) (*container.ExecResult, error) {
	if len(opts.Command) == 0 {
		return nil, pkgerrors.InvalidArgument("command is required", nil)
	}
	env := make([]string, 0, len(opts.Env))
	for k, v := range opts.Env {
		env = append(env, k+"="+v)
	}
	create, err := r.client.ContainerExecCreate(ctx, containerID, dockerccontainer.ExecOptions{
		Cmd:          opts.Command,
		Env:          env,
		WorkingDir:   opts.WorkDir,
		User:         opts.User,
		Tty:          opts.Tty,
		AttachStdout: true,
		AttachStderr: true,
	})
	if err != nil {
		if client.IsErrNotFound(err) {
			return nil, container.ErrContainerNotFound
		}
		return nil, pkgerrors.Internal("failed to create exec", err)
	}
	hijack, err := r.client.ContainerExecAttach(ctx, create.ID, dockerccontainer.ExecAttachOptions{Tty: opts.Tty})
	if err != nil {
		return nil, pkgerrors.Internal("failed to attach exec", err)
	}
	defer hijack.Close()
	out, _ := io.ReadAll(hijack.Reader)
	insp, err := r.client.ContainerExecInspect(ctx, create.ID)
	if err != nil {
		return nil, pkgerrors.Internal("failed to inspect exec", err)
	}
	return &container.ExecResult{
		ExitCode: insp.ExitCode,
		Stdout:   string(out),
	}, nil
}

func (r *Runtime) Wait(ctx context.Context, containerID string) (int, error) {
	statusCh, errCh := r.client.ContainerWait(ctx, containerID, dockerccontainer.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			return -1, pkgerrors.Internal("failed waiting for container", err)
		}
	case st := <-statusCh:
		return int(st.StatusCode), nil
	case <-ctx.Done():
		return -1, ctx.Err()
	}
	return -1, ctx.Err()
}

func (r *Runtime) Stats(ctx context.Context, containerID string) (*container.ContainerStats, error) {
	reader, err := r.client.ContainerStatsOneShot(ctx, containerID)
	if err != nil {
		if client.IsErrNotFound(err) {
			return nil, container.ErrContainerNotFound
		}
		return nil, pkgerrors.Internal("failed to get container stats", err)
	}
	defer reader.Body.Close()

	var stats dockerccontainer.StatsResponse
	if err := json.NewDecoder(reader.Body).Decode(&stats); err != nil {
		return nil, pkgerrors.Internal("failed to decode container stats", err)
	}

	cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage - stats.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(stats.CPUStats.SystemUsage - stats.PreCPUStats.SystemUsage)
	cpuPercent := 0.0
	if systemDelta > 0 && cpuDelta > 0 {
		online := float64(stats.CPUStats.OnlineCPUs)
		if online == 0 {
			online = float64(len(stats.CPUStats.CPUUsage.PercpuUsage))
		}
		if online == 0 {
			online = 1
		}
		cpuPercent = (cpuDelta / systemDelta) * online * 100.0
	}

	var rx, tx int64
	for _, n := range stats.Networks {
		rx += int64(n.RxBytes)
		tx += int64(n.TxBytes)
	}

	return &container.ContainerStats{
		CPUPercent:  cpuPercent,
		MemoryUsage: int64(stats.MemoryStats.Usage),
		MemoryLimit: int64(stats.MemoryStats.Limit),
		NetworkRx:   rx,
		NetworkTx:   tx,
		Timestamp:   time.Now(),
	}, nil
}

func mapInspect(insp dockerccontainer.InspectResponse) *container.Container {
	c := &container.Container{}
	if insp.ContainerJSONBase != nil {
		c.ID = insp.ID
		c.Name = strings.TrimPrefix(insp.Name, "/")
		if insp.Created != "" {
			if t, err := time.Parse(time.RFC3339Nano, insp.Created); err == nil {
				c.CreatedAt = t
			}
		}
		if insp.State != nil {
			c.State = mapDockerState(string(insp.State.Status))
			c.ExitCode = insp.State.ExitCode
			if t, err := time.Parse(time.RFC3339Nano, insp.State.StartedAt); err == nil {
				c.StartedAt = t
			}
			if t, err := time.Parse(time.RFC3339Nano, insp.State.FinishedAt); err == nil && !t.IsZero() {
				c.FinishedAt = t
			}
		}
		if insp.HostConfig != nil {
			c.Memory = insp.HostConfig.Memory / (1024 * 1024)
			if insp.HostConfig.NanoCPUs > 0 {
				c.CPU = float64(insp.HostConfig.NanoCPUs) / 1e9
			}
		}
	}
	if insp.Config != nil {
		c.Image = insp.Config.Image
		c.Labels = insp.Config.Labels
	}
	if insp.NetworkSettings != nil {
		for _, n := range insp.NetworkSettings.Networks {
			if n != nil && n.IPAddress != "" {
				c.IPAddress = n.IPAddress
				break
			}
		}
	}
	return c
}

func mapDockerState(s string) container.ContainerState {
	switch strings.ToLower(s) {
	case "created":
		return container.ContainerStateCreated
	case "running":
		return container.ContainerStateRunning
	case "paused":
		return container.ContainerStatePaused
	case "restarting":
		return container.ContainerStateRestarting
	case "exited":
		return container.ContainerStateExited
	case "dead":
		return container.ContainerStateDead
	default:
		return container.ContainerState(s)
	}
}

var _ container.ContainerRuntime = (*Runtime)(nil)

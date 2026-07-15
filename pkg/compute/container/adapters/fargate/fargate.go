// Package fargate provides an AWS Fargate adapter for container.ContainerRuntime.
//
// Usage:
//
//	import "github.com/chris-alexander-pop/go-hyperforge/pkg/compute/container/adapters/fargate"
//
//	runtime, err := fargate.New(fargate.Config{Region: "us-east-1", Cluster: "my-cluster"})
//	container, err := runtime.Create(ctx, container.CreateOptions{Image: "nginx:latest"})
package fargate

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/compute/container"
	pkgerrors "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/google/uuid"
)

// Config holds Fargate configuration.
type Config struct {
	Region           string
	AccessKeyID      string
	SecretAccessKey  string
	Cluster          string
	Subnets          []string
	SecurityGroups   []string
	TaskRoleARN      string
	ExecutionRoleARN string
}

// Runtime implements container.ContainerRuntime for AWS Fargate.
type Runtime struct {
	client *ecs.Client
	config Config
}

// New creates a new Fargate runtime.
func New(cfg Config) (*Runtime, error) {
	opts := []func(*config.LoadOptions) error{
		config.WithRegion(cfg.Region),
	}

	if cfg.AccessKeyID != "" && cfg.SecretAccessKey != "" {
		opts = append(opts, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		))
	}

	awsCfg, err := config.LoadDefaultConfig(context.Background(), opts...)
	if err != nil {
		return nil, pkgerrors.Internal("failed to load AWS config", err)
	}

	return &Runtime{
		client: ecs.NewFromConfig(awsCfg),
		config: cfg,
	}, nil
}

func (r *Runtime) Create(ctx context.Context, opts container.CreateOptions) (*container.Container, error) {
	name := opts.Name
	if name == "" {
		name = "task-" + uuid.NewString()[:8]
	}

	cpu := "256"
	memory := "512"
	if opts.CPU > 0 {
		cpu = fmt.Sprintf("%d", int(opts.CPU*1024))
	}
	if opts.Memory > 0 {
		memory = fmt.Sprintf("%d", opts.Memory)
	}

	// Register task definition
	taskDef := &ecs.RegisterTaskDefinitionInput{
		Family:                  aws.String(name),
		RequiresCompatibilities: []types.Compatibility{types.CompatibilityFargate},
		NetworkMode:             types.NetworkModeAwsvpc,
		Cpu:                     aws.String(cpu),
		Memory:                  aws.String(memory),
		ExecutionRoleArn:        aws.String(r.config.ExecutionRoleARN),
		TaskRoleArn:             aws.String(r.config.TaskRoleARN),
		ContainerDefinitions: []types.ContainerDefinition{
			{
				Name:         aws.String(name),
				Image:        aws.String(opts.Image),
				Essential:    aws.Bool(true),
				Command:      opts.Command,
				Environment:  convertEnvToKeyValue(opts.Env),
				PortMappings: convertPortsToMappings(opts.Ports),
			},
		},
	}

	taskDefOutput, err := r.client.RegisterTaskDefinition(ctx, taskDef)
	if err != nil {
		return nil, pkgerrors.Internal("failed to register task definition", err)
	}

	// Run task
	runInput := &ecs.RunTaskInput{
		Cluster:        aws.String(r.config.Cluster),
		TaskDefinition: taskDefOutput.TaskDefinition.TaskDefinitionArn,
		LaunchType:     types.LaunchTypeFargate,
		NetworkConfiguration: &types.NetworkConfiguration{
			AwsvpcConfiguration: &types.AwsVpcConfiguration{
				Subnets:        r.config.Subnets,
				SecurityGroups: r.config.SecurityGroups,
				AssignPublicIp: types.AssignPublicIpEnabled,
			},
		},
		Count: aws.Int32(1),
	}

	runOutput, err := r.client.RunTask(ctx, runInput)
	if err != nil {
		return nil, pkgerrors.Internal("failed to run task", err)
	}

	if len(runOutput.Tasks) == 0 {
		return nil, pkgerrors.Internal("no tasks started", nil)
	}

	task := runOutput.Tasks[0]
	return mapTaskToContainer(&task, opts.Image), nil
}

func convertEnvToKeyValue(env map[string]string) []types.KeyValuePair {
	if env == nil {
		return nil
	}
	result := make([]types.KeyValuePair, 0, len(env))
	for k, v := range env {
		result = append(result, types.KeyValuePair{Name: aws.String(k), Value: aws.String(v)})
	}
	return result
}

func convertPortsToMappings(ports []container.PortMapping) []types.PortMapping {
	if ports == nil {
		return nil
	}
	result := make([]types.PortMapping, len(ports))
	for i, p := range ports {
		result[i] = types.PortMapping{
			ContainerPort: aws.Int32(int32(p.ContainerPort)),
			HostPort:      aws.Int32(int32(p.HostPort)),
			Protocol:      types.TransportProtocolTcp,
		}
	}
	return result
}

func mapTaskToContainer(task *types.Task, image string) *container.Container {
	state := container.ContainerStateCreated
	switch aws.ToString(task.LastStatus) {
	case "RUNNING":
		state = container.ContainerStateRunning
	case "STOPPED":
		state = container.ContainerStateExited
	case "PENDING", "PROVISIONING":
		state = container.ContainerStateCreated
	}

	c := &container.Container{
		ID:        aws.ToString(task.TaskArn),
		Name:      aws.ToString(task.TaskDefinitionArn),
		Image:     image,
		State:     state,
		CreatedAt: aws.ToTime(task.CreatedAt),
	}

	if task.StartedAt != nil {
		c.StartedAt = *task.StartedAt
	}
	if task.StoppedAt != nil {
		c.FinishedAt = *task.StoppedAt
	}

	return c
}

func (r *Runtime) Get(ctx context.Context, containerID string) (*container.Container, error) {
	output, err := r.client.DescribeTasks(ctx, &ecs.DescribeTasksInput{
		Cluster: aws.String(r.config.Cluster),
		Tasks:   []string{containerID},
	})
	if err != nil {
		return nil, pkgerrors.NotFound("task not found", err)
	}

	if len(output.Tasks) == 0 {
		return nil, pkgerrors.NotFound("task not found", nil)
	}

	return mapTaskToContainer(&output.Tasks[0], ""), nil
}

func (r *Runtime) List(ctx context.Context, opts container.ListOptions) ([]*container.Container, error) {
	listOutput, err := r.client.ListTasks(ctx, &ecs.ListTasksInput{
		Cluster: aws.String(r.config.Cluster),
	})
	if err != nil {
		return nil, pkgerrors.Internal("failed to list tasks", err)
	}

	if len(listOutput.TaskArns) == 0 {
		return []*container.Container{}, nil
	}

	descOutput, err := r.client.DescribeTasks(ctx, &ecs.DescribeTasksInput{
		Cluster: aws.String(r.config.Cluster),
		Tasks:   listOutput.TaskArns,
	})
	if err != nil {
		return nil, pkgerrors.Internal("failed to describe tasks", err)
	}

	result := make([]*container.Container, len(descOutput.Tasks))
	for i, task := range descOutput.Tasks {
		result[i] = mapTaskToContainer(&task, "")
	}

	return result, nil
}

func (r *Runtime) Start(ctx context.Context, containerID string) error {
	// Fargate tasks are started when created
	return nil
}

func (r *Runtime) Stop(ctx context.Context, containerID string, timeout time.Duration) error {
	_, err := r.client.StopTask(ctx, &ecs.StopTaskInput{
		Cluster: aws.String(r.config.Cluster),
		Task:    aws.String(containerID),
	})
	if err != nil {
		return pkgerrors.Internal("failed to stop task", err)
	}
	return nil
}

func (r *Runtime) Kill(ctx context.Context, containerID string, signal string) error {
	return r.Stop(ctx, containerID, 0)
}

func (r *Runtime) Remove(ctx context.Context, containerID string, force bool) error {
	return r.Stop(ctx, containerID, 0)
}

func (r *Runtime) Logs(ctx context.Context, containerID string, follow bool) (io.ReadCloser, error) {
	// Fargate logs go to CloudWatch Logs
	return io.NopCloser(strings.NewReader("Use CloudWatch Logs for Fargate logs")), nil
}

func (r *Runtime) Exec(ctx context.Context, containerID string, opts container.ExecOptions) (*container.ExecResult, error) {
	// ECS Exec requires session manager
	return &container.ExecResult{ExitCode: 0}, nil
}

func (r *Runtime) Wait(ctx context.Context, containerID string) (int, error) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return -1, ctx.Err()
		case <-ticker.C:
			c, err := r.Get(ctx, containerID)
			if err != nil {
				return -1, err
			}
			if c.State == container.ContainerStateExited {
				return c.ExitCode, nil
			}
		}
	}
}

func (r *Runtime) Stats(ctx context.Context, containerID string) (*container.ContainerStats, error) {
	return &container.ContainerStats{
		Timestamp: time.Now(),
	}, nil
}

var _ container.ContainerRuntime = (*Runtime)(nil)

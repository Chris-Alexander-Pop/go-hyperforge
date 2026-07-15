package ec2

import (
	"context"
	"encoding/base64"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awsec2 "github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/compute/vm"
	pkgerrors "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
)

// Config holds EC2 adapter configuration with env tags matching vm.Config.
type Config struct {
	Region          string `env:"VM_AWS_REGION" env-default:"us-east-1"`
	AccessKeyID     string `env:"VM_AWS_ACCESS_KEY"`
	SecretAccessKey string `env:"VM_AWS_SECRET_KEY"`
	Endpoint        string `env:"VM_AWS_ENDPOINT"` // LocalStack / custom
	DefaultType     string `env:"VM_DEFAULT_TYPE" env-default:"t3.medium"`
}

// Manager implements vm.VMManager for AWS EC2.
type Manager struct {
	client      EC2API
	config      Config
	defaultType string
}

// New creates an EC2 manager using the AWS SDK default credential chain.
func New(cfg Config) (*Manager, error) {
	opts := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(cfg.Region),
	}
	if cfg.AccessKeyID != "" && cfg.SecretAccessKey != "" {
		opts = append(opts, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		))
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(), opts...)
	if err != nil {
		return nil, pkgerrors.Internal("failed to load AWS config", err)
	}

	clientOpts := []func(*awsec2.Options){}
	if cfg.Endpoint != "" {
		clientOpts = append(clientOpts, func(o *awsec2.Options) {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
		})
	}

	return NewWithClient(awsec2.NewFromConfig(awsCfg, clientOpts...), cfg), nil
}

// NewWithClient creates a Manager with an injected EC2API (production SDK or mock).
func NewWithClient(client EC2API, cfg Config) *Manager {
	dt := cfg.DefaultType
	if dt == "" {
		dt = "t3.medium"
	}
	return &Manager{client: client, config: cfg, defaultType: dt}
}

func (m *Manager) Create(ctx context.Context, opts vm.CreateOptions) (*vm.Instance, error) {
	if opts.ImageID == "" {
		return nil, pkgerrors.InvalidArgument("image_id is required", nil)
	}

	instanceType := opts.InstanceType
	if instanceType == "" {
		instanceType = m.defaultType
	}

	input := &awsec2.RunInstancesInput{
		ImageId:      aws.String(opts.ImageID),
		InstanceType: types.InstanceType(instanceType),
		MinCount:     aws.Int32(1),
		MaxCount:     aws.Int32(1),
	}
	if opts.KeyName != "" {
		input.KeyName = aws.String(opts.KeyName)
	}
	if opts.SubnetID != "" {
		input.SubnetId = aws.String(opts.SubnetID)
	}
	if len(opts.SecurityGroupIDs) > 0 {
		input.SecurityGroupIds = opts.SecurityGroupIDs
	}
	if opts.UserData != "" {
		input.UserData = aws.String(base64.StdEncoding.EncodeToString([]byte(opts.UserData)))
	}
	if opts.Zone != "" {
		input.Placement = &types.Placement{AvailabilityZone: aws.String(opts.Zone)}
	}

	tags := make([]types.Tag, 0, len(opts.Tags)+1)
	if opts.Name != "" {
		tags = append(tags, types.Tag{Key: aws.String("Name"), Value: aws.String(opts.Name)})
	}
	for k, v := range opts.Tags {
		tags = append(tags, types.Tag{Key: aws.String(k), Value: aws.String(v)})
	}
	if len(tags) > 0 {
		input.TagSpecifications = []types.TagSpecification{{
			ResourceType: types.ResourceTypeInstance,
			Tags:         tags,
		}}
	}

	out, err := m.client.RunInstances(ctx, input)
	if err != nil {
		return nil, pkgerrors.Internal("failed to run EC2 instance", err)
	}
	if len(out.Instances) == 0 {
		return nil, pkgerrors.Internal("RunInstances returned no instances", nil)
	}
	return mapInstance(&out.Instances[0]), nil
}

func (m *Manager) Get(ctx context.Context, instanceID string) (*vm.Instance, error) {
	out, err := m.client.DescribeInstances(ctx, &awsec2.DescribeInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil {
		return nil, pkgerrors.Internal("failed to describe EC2 instance", err)
	}
	inst := firstInstance(out)
	if inst == nil {
		return nil, vm.ErrInstanceNotFound
	}
	return mapInstance(inst), nil
}

func (m *Manager) List(ctx context.Context, opts vm.ListOptions) (*vm.ListResult, error) {
	input := &awsec2.DescribeInstancesInput{}
	if opts.PageToken != "" {
		input.NextToken = aws.String(opts.PageToken)
	}
	if opts.State != "" {
		input.Filters = append(input.Filters, types.Filter{
			Name:   aws.String("instance-state-name"),
			Values: []string{string(opts.State)},
		})
	}
	for k, v := range opts.Tags {
		input.Filters = append(input.Filters, types.Filter{
			Name:   aws.String("tag:" + k),
			Values: []string{v},
		})
	}

	out, err := m.client.DescribeInstances(ctx, input)
	if err != nil {
		return nil, pkgerrors.Internal("failed to list EC2 instances", err)
	}

	result := &vm.ListResult{Instances: make([]*vm.Instance, 0)}
	for _, res := range out.Reservations {
		for i := range res.Instances {
			result.Instances = append(result.Instances, mapInstance(&res.Instances[i]))
		}
	}
	if opts.Limit > 0 && len(result.Instances) > opts.Limit {
		result.Instances = result.Instances[:opts.Limit]
	}
	if out.NextToken != nil {
		result.NextPageToken = *out.NextToken
	}
	return result, nil
}

func (m *Manager) Start(ctx context.Context, instanceID string) error {
	_, err := m.client.StartInstances(ctx, &awsec2.StartInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil {
		return pkgerrors.Internal("failed to start EC2 instance", err)
	}
	return nil
}

func (m *Manager) Stop(ctx context.Context, instanceID string) error {
	_, err := m.client.StopInstances(ctx, &awsec2.StopInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil {
		return pkgerrors.Internal("failed to stop EC2 instance", err)
	}
	return nil
}

func (m *Manager) Reboot(ctx context.Context, instanceID string) error {
	_, err := m.client.RebootInstances(ctx, &awsec2.RebootInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil {
		return pkgerrors.Internal("failed to reboot EC2 instance", err)
	}
	return nil
}

func (m *Manager) Terminate(ctx context.Context, instanceID string) error {
	_, err := m.client.TerminateInstances(ctx, &awsec2.TerminateInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil {
		return pkgerrors.Internal("failed to terminate EC2 instance", err)
	}
	return nil
}

func (m *Manager) UpdateTags(ctx context.Context, instanceID string, tags map[string]string) error {
	if len(tags) == 0 {
		return nil
	}
	ec2Tags := make([]types.Tag, 0, len(tags))
	for k, v := range tags {
		ec2Tags = append(ec2Tags, types.Tag{Key: aws.String(k), Value: aws.String(v)})
	}
	_, err := m.client.CreateTags(ctx, &awsec2.CreateTagsInput{
		Resources: []string{instanceID},
		Tags:      ec2Tags,
	})
	if err != nil {
		return pkgerrors.Internal("failed to update EC2 tags", err)
	}
	return nil
}

func (m *Manager) GetConsoleOutput(ctx context.Context, instanceID string) (string, error) {
	out, err := m.client.GetConsoleOutput(ctx, &awsec2.GetConsoleOutputInput{
		InstanceId: aws.String(instanceID),
	})
	if err != nil {
		return "", pkgerrors.Internal("failed to get console output", err)
	}
	raw := aws.ToString(out.Output)
	if raw == "" {
		return "", nil
	}
	decoded, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return raw, nil
	}
	return string(decoded), nil
}

func firstInstance(out *awsec2.DescribeInstancesOutput) *types.Instance {
	if out == nil {
		return nil
	}
	for _, res := range out.Reservations {
		if len(res.Instances) > 0 {
			return &res.Instances[0]
		}
	}
	return nil
}

func mapInstance(inst *types.Instance) *vm.Instance {
	state := vm.InstanceStatePending
	if inst.State != nil {
		switch inst.State.Name {
		case types.InstanceStateNamePending:
			state = vm.InstanceStatePending
		case types.InstanceStateNameRunning:
			state = vm.InstanceStateRunning
		case types.InstanceStateNameStopping:
			state = vm.InstanceStateStopping
		case types.InstanceStateNameStopped:
			state = vm.InstanceStateStopped
		case types.InstanceStateNameShuttingDown:
			state = vm.InstanceStateTerminating
		case types.InstanceStateNameTerminated:
			state = vm.InstanceStateTerminated
		}
	}

	tags := make(map[string]string, len(inst.Tags))
	name := ""
	for _, t := range inst.Tags {
		k, v := aws.ToString(t.Key), aws.ToString(t.Value)
		tags[k] = v
		if k == "Name" {
			name = v
		}
	}

	sgs := make([]string, 0, len(inst.SecurityGroups))
	for _, sg := range inst.SecurityGroups {
		sgs = append(sgs, aws.ToString(sg.GroupId))
	}

	launch := time.Time{}
	if inst.LaunchTime != nil {
		launch = *inst.LaunchTime
	}

	zone := ""
	if inst.Placement != nil {
		zone = aws.ToString(inst.Placement.AvailabilityZone)
	}

	return &vm.Instance{
		ID:             aws.ToString(inst.InstanceId),
		Name:           name,
		State:          state,
		InstanceType:   string(inst.InstanceType),
		ImageID:        aws.ToString(inst.ImageId),
		PublicIP:       aws.ToString(inst.PublicIpAddress),
		PrivateIP:      aws.ToString(inst.PrivateIpAddress),
		Zone:           zone,
		VPCSubnetID:    aws.ToString(inst.SubnetId),
		SecurityGroups: sgs,
		Tags:           tags,
		LaunchTime:     launch,
	}
}

var _ vm.VMManager = (*Manager)(nil)

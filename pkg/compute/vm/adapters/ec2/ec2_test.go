package ec2

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsec2 "github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/chris-alexander-pop/system-design-library/pkg/compute/vm"
	pkgerrors "github.com/chris-alexander-pop/system-design-library/pkg/errors"
)

type mockEC2 struct {
	runOut        *awsec2.RunInstancesOutput
	runErr        error
	describeOut   *awsec2.DescribeInstancesOutput
	describeErr   error
	startErr      error
	stopErr       error
	terminateErr  error
	rebootErr     error
	createTagsErr error
	consoleOut    *awsec2.GetConsoleOutputOutput
	consoleErr    error

	lastRun *awsec2.RunInstancesInput
}

func (m *mockEC2) RunInstances(ctx context.Context, params *awsec2.RunInstancesInput, optFns ...func(*awsec2.Options)) (*awsec2.RunInstancesOutput, error) {
	m.lastRun = params
	return m.runOut, m.runErr
}
func (m *mockEC2) DescribeInstances(ctx context.Context, params *awsec2.DescribeInstancesInput, optFns ...func(*awsec2.Options)) (*awsec2.DescribeInstancesOutput, error) {
	return m.describeOut, m.describeErr
}
func (m *mockEC2) StartInstances(ctx context.Context, params *awsec2.StartInstancesInput, optFns ...func(*awsec2.Options)) (*awsec2.StartInstancesOutput, error) {
	return &awsec2.StartInstancesOutput{}, m.startErr
}
func (m *mockEC2) StopInstances(ctx context.Context, params *awsec2.StopInstancesInput, optFns ...func(*awsec2.Options)) (*awsec2.StopInstancesOutput, error) {
	return &awsec2.StopInstancesOutput{}, m.stopErr
}
func (m *mockEC2) TerminateInstances(ctx context.Context, params *awsec2.TerminateInstancesInput, optFns ...func(*awsec2.Options)) (*awsec2.TerminateInstancesOutput, error) {
	return &awsec2.TerminateInstancesOutput{}, m.terminateErr
}
func (m *mockEC2) RebootInstances(ctx context.Context, params *awsec2.RebootInstancesInput, optFns ...func(*awsec2.Options)) (*awsec2.RebootInstancesOutput, error) {
	return &awsec2.RebootInstancesOutput{}, m.rebootErr
}
func (m *mockEC2) CreateTags(ctx context.Context, params *awsec2.CreateTagsInput, optFns ...func(*awsec2.Options)) (*awsec2.CreateTagsOutput, error) {
	return &awsec2.CreateTagsOutput{}, m.createTagsErr
}
func (m *mockEC2) GetConsoleOutput(ctx context.Context, params *awsec2.GetConsoleOutputInput, optFns ...func(*awsec2.Options)) (*awsec2.GetConsoleOutputOutput, error) {
	return m.consoleOut, m.consoleErr
}

func sampleInstance(id string) types.Instance {
	now := time.Now()
	return types.Instance{
		InstanceId:       aws.String(id),
		ImageId:          aws.String("ami-123"),
		InstanceType:     types.InstanceTypeT3Medium,
		PublicIpAddress:  aws.String("1.2.3.4"),
		PrivateIpAddress: aws.String("10.0.0.5"),
		SubnetId:         aws.String("subnet-1"),
		LaunchTime:       &now,
		State:            &types.InstanceState{Name: types.InstanceStateNameRunning},
		Placement:        &types.Placement{AvailabilityZone: aws.String("us-east-1a")},
		SecurityGroups:   []types.GroupIdentifier{{GroupId: aws.String("sg-1")}},
		Tags:             []types.Tag{{Key: aws.String("Name"), Value: aws.String("web")}},
	}
}

func TestCreateGetListLifecycle(t *testing.T) {
	inst := sampleInstance("i-abc")
	mock := &mockEC2{
		runOut: &awsec2.RunInstancesOutput{Instances: []types.Instance{inst}},
		describeOut: &awsec2.DescribeInstancesOutput{
			Reservations: []types.Reservation{{Instances: []types.Instance{inst}}},
		},
	}
	mgr := NewWithClient(mock, Config{DefaultType: "t3.small"})
	ctx := context.Background()

	created, err := mgr.Create(ctx, vm.CreateOptions{
		Name: "web", ImageID: "ami-123", InstanceType: "t3.medium",
		Tags: map[string]string{"env": "test"},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.ID != "i-abc" || created.State != vm.InstanceStateRunning {
		t.Fatalf("unexpected instance: %+v", created)
	}
	if mock.lastRun == nil || aws.ToString(mock.lastRun.ImageId) != "ami-123" {
		t.Fatalf("RunInstances not called correctly: %+v", mock.lastRun)
	}

	got, err := mgr.Get(ctx, "i-abc")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Name != "web" || got.PublicIP != "1.2.3.4" {
		t.Fatalf("unexpected get: %+v", got)
	}

	list, err := mgr.List(ctx, vm.ListOptions{})
	if err != nil || len(list.Instances) != 1 {
		t.Fatalf("List: %v len=%d", err, len(list.Instances))
	}

	for _, op := range []func() error{
		func() error { return mgr.Start(ctx, "i-abc") },
		func() error { return mgr.Stop(ctx, "i-abc") },
		func() error { return mgr.Reboot(ctx, "i-abc") },
		func() error { return mgr.UpdateTags(ctx, "i-abc", map[string]string{"a": "b"}) },
		func() error { return mgr.Terminate(ctx, "i-abc") },
	} {
		if err := op(); err != nil {
			t.Fatalf("lifecycle op: %v", err)
		}
	}
}

func TestCreateRequiresImage(t *testing.T) {
	mgr := NewWithClient(&mockEC2{}, Config{})
	_, err := mgr.Create(context.Background(), vm.CreateOptions{})
	if err == nil || !pkgerrors.IsCode(err, pkgerrors.CodeInvalidArgument) {
		t.Fatalf("expected InvalidArgument, got %v", err)
	}
}

func TestGetNotFound(t *testing.T) {
	mgr := NewWithClient(&mockEC2{
		describeOut: &awsec2.DescribeInstancesOutput{},
	}, Config{})
	_, err := mgr.Get(context.Background(), "i-missing")
	if err != vm.ErrInstanceNotFound {
		t.Fatalf("expected ErrInstanceNotFound, got %v", err)
	}
}

func TestMapInstanceStates(t *testing.T) {
	cases := []struct {
		name string
		in   types.InstanceStateName
		want vm.InstanceState
	}{
		{"pending", types.InstanceStateNamePending, vm.InstanceStatePending},
		{"running", types.InstanceStateNameRunning, vm.InstanceStateRunning},
		{"stopping", types.InstanceStateNameStopping, vm.InstanceStateStopping},
		{"stopped", types.InstanceStateNameStopped, vm.InstanceStateStopped},
		{"shutting", types.InstanceStateNameShuttingDown, vm.InstanceStateTerminating},
		{"terminated", types.InstanceStateNameTerminated, vm.InstanceStateTerminated},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := mapInstance(&types.Instance{
				InstanceId: aws.String("i-1"),
				State:      &types.InstanceState{Name: tc.in},
			})
			if got.State != tc.want {
				t.Fatalf("got %s want %s", got.State, tc.want)
			}
		})
	}
}

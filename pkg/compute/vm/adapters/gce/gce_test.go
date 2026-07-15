package gce

import (
	"context"
	"errors"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/compute/vm"
	compute "google.golang.org/api/compute/v1"
)

type mockAPI struct {
	inst    *compute.Instance
	list    []*compute.Instance
	getErr  error
	opErr   error
	console string
}

func (m *mockAPI) Insert(ctx context.Context, project, zone string, inst *compute.Instance) (*compute.Operation, error) {
	m.inst = inst
	return &compute.Operation{Status: "DONE"}, m.opErr
}
func (m *mockAPI) Get(ctx context.Context, project, zone, name string) (*compute.Instance, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	if m.inst == nil || m.inst.Name != name {
		return nil, errors.New("404 not found")
	}
	return m.inst, nil
}
func (m *mockAPI) List(ctx context.Context, project, zone string) ([]*compute.Instance, error) {
	return m.list, m.opErr
}
func (m *mockAPI) Start(ctx context.Context, project, zone, name string) (*compute.Operation, error) {
	return &compute.Operation{}, m.opErr
}
func (m *mockAPI) Stop(ctx context.Context, project, zone, name string) (*compute.Operation, error) {
	return &compute.Operation{}, m.opErr
}
func (m *mockAPI) Reset(ctx context.Context, project, zone, name string) (*compute.Operation, error) {
	return &compute.Operation{}, m.opErr
}
func (m *mockAPI) Delete(ctx context.Context, project, zone, name string) (*compute.Operation, error) {
	return &compute.Operation{}, m.opErr
}
func (m *mockAPI) SetLabels(ctx context.Context, project, zone, name string, req *compute.InstancesSetLabelsRequest) (*compute.Operation, error) {
	if m.inst != nil {
		m.inst.Labels = req.Labels
	}
	return &compute.Operation{}, m.opErr
}
func (m *mockAPI) GetSerialPortOutput(ctx context.Context, project, zone, name string) (string, error) {
	return m.console, m.opErr
}

func TestGCECreateGetLifecycle(t *testing.T) {
	api := &mockAPI{}
	mgr := NewWithAPI(api, "proj", "us-central1-a", "e2-medium")
	ctx := context.Background()

	created, err := mgr.Create(ctx, vm.CreateOptions{
		Name: "web-1", ImageID: "projects/debian-cloud/global/images/family/debian-12",
		Tags: map[string]string{"Env": "dev"},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.ID != "web-1" {
		t.Fatalf("expected web-1, got %s", created.ID)
	}
	api.inst.Status = "RUNNING"
	api.inst.NetworkInterfaces = []*compute.NetworkInterface{{
		NetworkIP:     "10.0.0.2",
		AccessConfigs: []*compute.AccessConfig{{NatIP: "8.8.8.8"}},
	}}

	got, err := mgr.Get(ctx, "web-1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.State != vm.InstanceStateRunning || got.PublicIP != "8.8.8.8" {
		t.Fatalf("unexpected: %+v", got)
	}

	api.list = []*compute.Instance{api.inst}
	list, err := mgr.List(ctx, vm.ListOptions{State: vm.InstanceStateRunning})
	if err != nil || len(list.Instances) != 1 {
		t.Fatalf("List: %v len=%d", err, len(list.Instances))
	}

	if err := mgr.Start(ctx, "web-1"); err != nil {
		t.Fatal(err)
	}
	if err := mgr.Stop(ctx, "web-1"); err != nil {
		t.Fatal(err)
	}
	if err := mgr.Reboot(ctx, "web-1"); err != nil {
		t.Fatal(err)
	}
	if err := mgr.UpdateTags(ctx, "web-1", map[string]string{"tier": "web"}); err != nil {
		t.Fatal(err)
	}
	api.console = "boot ok"
	out, err := mgr.GetConsoleOutput(ctx, "web-1")
	if err != nil || out != "boot ok" {
		t.Fatalf("console: %v %q", err, out)
	}
	if err := mgr.Terminate(ctx, "web-1"); err != nil {
		t.Fatal(err)
	}
}

func TestGCEGetNotFound(t *testing.T) {
	mgr := NewWithAPI(&mockAPI{getErr: errors.New("googleapi: Error 404: notFound")}, "p", "z", "")
	_, err := mgr.Get(context.Background(), "missing")
	if err != vm.ErrInstanceNotFound {
		t.Fatalf("expected not found, got %v", err)
	}
}

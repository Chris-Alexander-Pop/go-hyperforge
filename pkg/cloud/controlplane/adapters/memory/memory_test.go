package memory_test

import (
	"context"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/cloud"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/cloud/controlplane"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/cloud/controlplane/adapters/memory"
)

func TestControlPlaneRegisterGetList(t *testing.T) {
	cp := memory.New()
	ctx := context.Background()

	host := cloud.Host{
		ID:     "host-1",
		Name:   "node-a",
		Status: cloud.HostStatusReady,
		Capacity: cloud.Resources{
			VCPUs:    8,
			MemoryMB: 16384,
			DiskGB:   500,
		},
		Zone: "zone-a",
	}

	if err := cp.RegisterHost(ctx, host); err != nil {
		t.Fatalf("RegisterHost: %v", err)
	}

	got, err := cp.GetHost(ctx, "host-1")
	if err != nil {
		t.Fatalf("GetHost: %v", err)
	}
	if got.Name != "node-a" || got.Status != cloud.HostStatusReady {
		t.Fatalf("unexpected host: %+v", got)
	}

	list, err := cp.ListHosts(ctx)
	if err != nil {
		t.Fatalf("ListHosts: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 host, got %d", len(list))
	}
}

func TestControlPlaneDuplicateAndMissing(t *testing.T) {
	cp := memory.New()
	ctx := context.Background()

	host := cloud.Host{ID: "h1", Status: cloud.HostStatusReady}
	if err := cp.RegisterHost(ctx, host); err != nil {
		t.Fatalf("RegisterHost: %v", err)
	}
	if err := cp.RegisterHost(ctx, host); err != controlplane.ErrHostAlreadyRegistered {
		t.Fatalf("expected ErrHostAlreadyRegistered, got %v", err)
	}
	if _, err := cp.GetHost(ctx, "missing"); err != controlplane.ErrHostNotFound {
		t.Fatalf("expected ErrHostNotFound, got %v", err)
	}
}

func TestControlPlaneUpdateStatusAndDeregister(t *testing.T) {
	cp := memory.New()
	ctx := context.Background()

	if err := cp.RegisterHost(ctx, cloud.Host{ID: "h1", Status: cloud.HostStatusReady}); err != nil {
		t.Fatalf("RegisterHost: %v", err)
	}
	if err := cp.UpdateHostStatus(ctx, "h1", cloud.HostStatusMaintenance); err != nil {
		t.Fatalf("UpdateHostStatus: %v", err)
	}
	got, err := cp.GetHost(ctx, "h1")
	if err != nil {
		t.Fatalf("GetHost: %v", err)
	}
	if got.Status != cloud.HostStatusMaintenance {
		t.Fatalf("expected maintenance, got %s", got.Status)
	}
	if err := cp.DeregisterHost(ctx, "h1"); err != nil {
		t.Fatalf("DeregisterHost: %v", err)
	}
	if err := cp.DeregisterHost(ctx, "h1"); err != controlplane.ErrHostNotFound {
		t.Fatalf("expected ErrHostNotFound on second deregister, got %v", err)
	}
	if err := cp.UpdateHostStatus(ctx, "h1", cloud.HostStatusReady); err != controlplane.ErrHostNotFound {
		t.Fatalf("expected ErrHostNotFound on update, got %v", err)
	}
}

func TestControlPlaneCreateBindInstance(t *testing.T) {
	cp := memory.New()
	ctx := context.Background()

	if err := cp.RegisterHost(ctx, cloud.Host{
		ID: "h1", Status: cloud.HostStatusReady,
		Capacity: cloud.Resources{VCPUs: 4, MemoryMB: 8192, DiskGB: 100},
	}); err != nil {
		t.Fatal(err)
	}

	inst, err := cp.CreateInstance(ctx, controlplane.CreateInstanceRequest{
		Name: "vm-1", HostID: "h1",
		Resources: cloud.Resources{VCPUs: 2, MemoryMB: 4096, DiskGB: 20},
		Image:     "debian",
	})
	if err != nil {
		t.Fatalf("CreateInstance: %v", err)
	}
	if inst.HostID != "h1" || inst.Status != cloud.InstanceStatusProvisioning {
		t.Fatalf("unexpected instance: %+v", inst)
	}

	host, err := cp.GetHost(ctx, "h1")
	if err != nil {
		t.Fatal(err)
	}
	if host.Available.VCPUs != 2 || host.Available.MemoryMB != 4096 {
		t.Fatalf("capacity not reserved: %+v", host.Available)
	}

	got, err := cp.GetInstance(ctx, inst.ID)
	if err != nil || got.Name != "vm-1" {
		t.Fatalf("GetInstance: %v %+v", err, got)
	}

	unbound, err := cp.CreateInstance(ctx, controlplane.CreateInstanceRequest{
		Name: "vm-2", Resources: cloud.Resources{VCPUs: 1, MemoryMB: 1024},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := cp.BindInstance(ctx, unbound.ID, "h1"); err != nil {
		t.Fatal(err)
	}
	if err := cp.BindInstance(ctx, unbound.ID, "h1"); err != controlplane.ErrInstanceAlreadyBound {
		t.Fatalf("expected already bound, got %v", err)
	}

	list, err := cp.ListInstances(ctx, controlplane.ListInstancesOptions{HostID: "h1"})
	if err != nil || len(list) != 2 {
		t.Fatalf("ListInstances: %v len=%d", err, len(list))
	}

	if err := cp.UpdateInstanceStatus(ctx, inst.ID, cloud.InstanceStatusRunning); err != nil {
		t.Fatal(err)
	}
	if err := cp.UnbindInstance(ctx, inst.ID); err != nil {
		t.Fatal(err)
	}
	host, _ = cp.GetHost(ctx, "h1")
	if host.Available.VCPUs != 3 { // 4 - 1 (vm-2 still bound)
		t.Fatalf("expected 3 vcpus available after unbind, got %d", host.Available.VCPUs)
	}
	if err := cp.DeregisterHost(ctx, "h1"); err != controlplane.ErrHostHasInstances {
		t.Fatalf("expected ErrHostHasInstances, got %v", err)
	}
	if err := cp.DeleteInstance(ctx, unbound.ID); err != nil {
		t.Fatal(err)
	}
	if err := cp.DeleteInstance(ctx, inst.ID); err != nil {
		t.Fatal(err)
	}
	if err := cp.DeregisterHost(ctx, "h1"); err != nil {
		t.Fatal(err)
	}
}

func TestControlPlaneCapacityExhausted(t *testing.T) {
	cp := memory.New()
	ctx := context.Background()
	_ = cp.RegisterHost(ctx, cloud.Host{
		ID: "h1", Status: cloud.HostStatusReady,
		Capacity: cloud.Resources{VCPUs: 1, MemoryMB: 512, DiskGB: 10},
	})
	_, err := cp.CreateInstance(ctx, controlplane.CreateInstanceRequest{
		Name: "big", HostID: "h1",
		Resources: cloud.Resources{VCPUs: 8, MemoryMB: 4096},
	})
	if err != controlplane.ErrHostCapacityExhausted {
		t.Fatalf("expected capacity exhausted, got %v", err)
	}
}

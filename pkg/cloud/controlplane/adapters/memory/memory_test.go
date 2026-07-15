package memory_test

import (
	"context"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/cloud"
	"github.com/chris-alexander-pop/system-design-library/pkg/cloud/controlplane"
	"github.com/chris-alexander-pop/system-design-library/pkg/cloud/controlplane/adapters/memory"
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

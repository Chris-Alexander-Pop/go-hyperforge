package memory_test

import (
	"context"
	"testing"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/cloud"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/cloud/provisioning"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/cloud/provisioning/adapters/memory"
)

func TestProvisionerLifecycle(t *testing.T) {
	p := memory.New().(*memory.MemoryProvisioner)
	ctx := context.Background()

	p.AddHost("bare-1", cloud.HostStatusOffline)

	if err := p.ProvisionHost(ctx, "bare-1", "http://images/ubuntu.iso"); err != nil {
		t.Fatalf("ProvisionHost: %v", err)
	}

	status, err := p.GetHostStatus(ctx, "bare-1")
	if err != nil {
		t.Fatalf("GetHostStatus: %v", err)
	}
	if status != cloud.HostStatusBusy && status != cloud.HostStatusReady {
		t.Fatalf("expected busy or ready during/after provision, got %s", status)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		status, err = p.GetHostStatus(ctx, "bare-1")
		if err != nil {
			t.Fatalf("GetHostStatus: %v", err)
		}
		if status == cloud.HostStatusReady {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if status != cloud.HostStatusReady {
		t.Fatalf("expected Ready after provision, got %s", status)
	}

	if err := p.PowerCycle(ctx, "bare-1"); err != nil {
		t.Fatalf("PowerCycle: %v", err)
	}

	if err := p.DeprovisionHost(ctx, "bare-1"); err != nil {
		t.Fatalf("DeprovisionHost: %v", err)
	}
	status, err = p.GetHostStatus(ctx, "bare-1")
	if err != nil {
		t.Fatalf("GetHostStatus after deprovision: %v", err)
	}
	if status != cloud.HostStatusOffline {
		t.Fatalf("expected Offline, got %s", status)
	}
}

func TestProvisionerNotFound(t *testing.T) {
	p := memory.New()
	ctx := context.Background()

	if err := p.ProvisionHost(ctx, "missing", "img"); err != provisioning.ErrHostNotFound {
		t.Fatalf("expected ErrHostNotFound, got %v", err)
	}
	if err := p.DeprovisionHost(ctx, "missing"); err != provisioning.ErrHostNotFound {
		t.Fatalf("expected ErrHostNotFound, got %v", err)
	}
	if _, err := p.GetHostStatus(ctx, "missing"); err != provisioning.ErrHostNotFound {
		t.Fatalf("expected ErrHostNotFound, got %v", err)
	}
	if err := p.PowerCycle(ctx, "missing"); err != provisioning.ErrHostNotFound {
		t.Fatalf("expected ErrHostNotFound, got %v", err)
	}
}

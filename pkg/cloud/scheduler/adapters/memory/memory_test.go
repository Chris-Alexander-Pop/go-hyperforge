package memory_test

import (
	"context"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/cloud"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/cloud/scheduler"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/cloud/scheduler/adapters/memory"
)

func largeHost(id, zone string, vcpus, mem int) cloud.Host {
	cap := cloud.Resources{VCPUs: vcpus, MemoryMB: mem, DiskGB: 1000}
	return cloud.Host{
		ID:        id,
		Status:    cloud.HostStatusReady,
		Zone:      zone,
		Capacity:  cap,
		Available: cap,
	}
}

func TestSchedulerNoHosts(t *testing.T) {
	s := memory.New()
	_, err := s.SelectHost(context.Background(), scheduler.Requirement{
		Resources: cloud.Resources{VCPUs: 1, MemoryMB: 512},
	})
	if err != scheduler.ErrNoHostFound {
		t.Fatalf("expected ErrNoHostFound, got %v", err)
	}
}

func TestSchedulerRandomFits(t *testing.T) {
	s := memory.New(scheduler.Config{Strategy: scheduler.StrategyRandom})
	s.AddHost(largeHost("a", "z1", 4, 8192))
	s.AddHost(largeHost("b", "z1", 4, 8192))

	id, err := s.SelectHost(context.Background(), scheduler.Requirement{
		Resources: cloud.Resources{VCPUs: 2, MemoryMB: 1024},
	})
	if err != nil {
		t.Fatalf("SelectHost: %v", err)
	}
	if id != "a" && id != "b" {
		t.Fatalf("unexpected host %q", id)
	}
}

func TestSchedulerBinpackPrefersFullerHost(t *testing.T) {
	s := memory.New(scheduler.Config{Strategy: scheduler.StrategyBinpack})

	// Host almost full vs mostly empty — binpack should pick the tighter fit.
	almostFull := largeHost("full", "z1", 8, 8192)
	almostFull.Available = cloud.Resources{VCPUs: 2, MemoryMB: 2048, DiskGB: 100}
	empty := largeHost("empty", "z1", 8, 8192)

	s.AddHost(almostFull)
	s.AddHost(empty)

	id, err := s.SelectHost(context.Background(), scheduler.Requirement{
		Resources: cloud.Resources{VCPUs: 1, MemoryMB: 512},
	})
	if err != nil {
		t.Fatalf("SelectHost: %v", err)
	}
	if id != "full" {
		t.Fatalf("binpack should prefer fuller host, got %q", id)
	}
}

func TestSchedulerSpreadDistributes(t *testing.T) {
	s := memory.New(scheduler.Config{Strategy: scheduler.StrategySpread})
	s.AddHost(largeHost("a", "z1", 16, 32768))
	s.AddHost(largeHost("b", "z1", 16, 32768))

	counts := map[string]int{}
	req := scheduler.Requirement{Resources: cloud.Resources{VCPUs: 1, MemoryMB: 256}}
	for i := 0; i < 10; i++ {
		id, err := s.SelectHost(context.Background(), req)
		if err != nil {
			t.Fatalf("SelectHost %d: %v", i, err)
		}
		counts[id]++
	}
	if counts["a"] == 0 || counts["b"] == 0 {
		t.Fatalf("spread should use both hosts, got %v", counts)
	}
	diff := counts["a"] - counts["b"]
	if diff < 0 {
		diff = -diff
	}
	if diff > 2 {
		t.Fatalf("spread imbalance too high: %v", counts)
	}
}

func TestSchedulerCapacityExhaustion(t *testing.T) {
	s := memory.New(scheduler.Config{Strategy: scheduler.StrategyBinpack})
	h := largeHost("tiny", "z1", 2, 1024)
	s.AddHost(h)

	req := scheduler.Requirement{Resources: cloud.Resources{VCPUs: 2, MemoryMB: 1024}}
	if _, err := s.SelectHost(context.Background(), req); err != nil {
		t.Fatalf("first SelectHost: %v", err)
	}
	if _, err := s.SelectHost(context.Background(), req); err != scheduler.ErrNoHostFound {
		t.Fatalf("expected ErrNoHostFound after capacity used, got %v", err)
	}
}

func TestSchedulerZoneFilter(t *testing.T) {
	s := memory.New()
	s.AddHost(largeHost("a", "zone-a", 8, 8192))
	s.AddHost(largeHost("b", "zone-b", 8, 8192))

	id, err := s.SelectHost(context.Background(), scheduler.Requirement{
		Resources: cloud.Resources{VCPUs: 1},
		Zone:      "zone-b",
	})
	if err != nil {
		t.Fatalf("SelectHost: %v", err)
	}
	if id != "b" {
		t.Fatalf("expected zone-b host, got %q", id)
	}
}

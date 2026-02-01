package memory

import (
	"context"
	"fmt"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/cloud"
	"github.com/chris-alexander-pop/system-design-library/pkg/cloud/hypervisor"
)

func BenchmarkCreateVM_DuplicateCheck(b *testing.B) {
	h := New()
	ctx := context.Background()

	// Pre-populate with 5000 VMs
	for i := 0; i < 5000; i++ {
		spec := hypervisor.VMSpec{
			Name:         fmt.Sprintf("vm-pre-%d", i),
			InstanceType: cloud.InstanceTypeSmall,
			Image:        "ubuntu-20.04",
		}
		_, err := h.CreateVM(ctx, spec)
		if err != nil {
			b.Fatalf("failed to create pre-population vm: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Use a unique name for each iteration to force the check
		spec := hypervisor.VMSpec{
			Name:         fmt.Sprintf("vm-bench-%d", i),
			InstanceType: cloud.InstanceTypeSmall,
			Image:        "ubuntu-20.04",
		}
		_, err := h.CreateVM(ctx, spec)
		if err != nil {
			b.Fatalf("failed to create vm: %v", err)
		}
	}
}

func TestDuplicateName(t *testing.T) {
	h := New()
	ctx := context.Background()

	spec := hypervisor.VMSpec{
		Name:         "test-vm",
		InstanceType: cloud.InstanceTypeSmall,
		Image:        "ubuntu-20.04",
	}

	// 1. Create VM
	id, err := h.CreateVM(ctx, spec)
	if err != nil {
		t.Fatalf("failed to create vm: %v", err)
	}

	// 2. Try to create duplicate
	_, err = h.CreateVM(ctx, spec)
	if err != hypervisor.ErrVMAlreadyExists {
		t.Errorf("expected ErrVMAlreadyExists, got %v", err)
	}

	// 3. Create another VM
	spec2 := spec
	spec2.Name = "test-vm-2"
	_, err = h.CreateVM(ctx, spec2)
	if err != nil {
		t.Fatalf("failed to create second vm: %v", err)
	}

	// 4. Delete first VM
	err = h.DeleteVM(ctx, id)
	if err != nil {
		t.Fatalf("failed to delete vm: %v", err)
	}

	// 5. Recreate first VM (reuse name)
	_, err = h.CreateVM(ctx, spec)
	if err != nil {
		t.Fatalf("failed to recreate vm with reused name: %v", err)
	}
}

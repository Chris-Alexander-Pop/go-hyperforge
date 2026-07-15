// Package hypervisor provides interfaces and adapters for managing Virtual Machines.
//
// Shipping: memory adapter only. Libvirt/QEMU/Firecracker drivers are reserved
// placeholders — not wired. For public-cloud VMs (EC2/GCE/Azure), see pkg/compute/vm.
//
// Basic usage:
//
//	import (
//		"github.com/chris-alexander-pop/system-design-library/pkg/cloud/hypervisor"
//		"github.com/chris-alexander-pop/system-design-library/pkg/cloud/hypervisor/adapters/memory"
//	)
//
//	hyp := memory.New()
//	id, err := hyp.CreateVM(ctx, hypervisor.VMSpec{
//		Name: "test-vm",
//		InstanceType: cloud.InstanceTypeSmall,
//	})
package hypervisor

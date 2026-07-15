// Package cloud provides interfaces and memory adapters for a private-cloud (IaaS) control plane.
//
// Domains covered (scaffold / memory-only today — production hypervisors and bare-metal
// backends are not wired; see root MISSING_CAPABILITIES.md):
//   - Hypervisor: VM lifecycle interface (+ memory adapter)
//   - Provisioning: bare-metal lifecycle interface (+ memory adapter)
//   - Scheduler: placement interface (+ memory adapter)
//   - Control Plane: API/state manager interface (+ memory adapter)
//
// Treat this package as a design surface for IaaS, not a ready Libvirt/QEMU/Firecracker stack.
package cloud

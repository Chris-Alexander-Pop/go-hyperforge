// Package cloud provides interfaces and memory adapters for a private-cloud (IaaS) control plane.
//
// Domains covered (scaffold / memory-only today — production hypervisors and bare-metal
// backends are not wired; see root MISSING_CAPABILITIES.md):
//   - Hypervisor: VM lifecycle interface (+ memory adapter)
//   - Provisioning: bare-metal lifecycle interface (+ memory adapter)
//   - Scheduler: placement with binpack / spread / random strategies (+ memory adapter)
//   - Control Plane: host inventory interface (+ memory adapter)
//
// Treat this package as a design surface for IaaS, not a ready Libvirt/QEMU/Firecracker stack.
//
// Relation to pkg/compute:
//
//   - pkg/cloud owns private-cloud host/instance vocabulary (Host, Resources,
//     InstanceStatus, InstanceType) for building or simulating an IaaS control plane.
//   - pkg/compute owns public-cloud / workload APIs (VM managers, container runtimes,
//     serverless) against AWS/GCP/Azure/Kubernetes. Those subpackages define their own
//     InstanceState and resource shapes for cloud-provider APIs.
//
// Prefer pkg/compute when calling managed cloud APIs; prefer pkg/cloud when modeling
// hypervisors, bare metal, and placement inside a private cloud.
package cloud

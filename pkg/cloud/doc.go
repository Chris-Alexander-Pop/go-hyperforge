// Package cloud provides interfaces and adapters for a private-cloud (IaaS) control plane.
//
// Domains:
//   - Hypervisor: memory, remote libvirt (JSON/HTTP), Firecracker (unix/HTTP API)
//   - Provisioning: memory, Redfish BMC, IPMI HTTP gateway
//   - Scheduler: placement with binpack / spread / random strategies (+ memory adapter)
//   - Control Plane: host inventory + instance create/bind APIs (+ memory adapter)
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

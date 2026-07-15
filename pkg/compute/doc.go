/*
Package compute provides public-cloud / workload compute abstractions.

Subpackages:

  - vm: Virtual machine lifecycle (memory, EC2, GCE; Azure VM scaffold)
  - container: Container runtime (memory, Docker, Kubernetes, Fargate)
  - serverless: FaaS (memory, Lambda, Cloud Functions, Azure Functions scaffold)

Relation to pkg/cloud:

  - pkg/compute targets managed cloud APIs and orchestrators (create a VM on
    AWS, run a pod on Kubernetes, invoke Lambda). Types such as InstanceState
    live under the subpackages that own the API surface.
  - pkg/cloud is a private-cloud / IaaS control-plane scaffold (hypervisor,
    bare-metal provisioning, placement scheduler, host inventory). Shared
    vocabulary there includes Host, Resources, and InstanceStatus.

Do not treat the two packages as interchangeable: compose compute adapters for
workload APIs, and cloud for building or simulating an IaaS control plane.
*/
package compute

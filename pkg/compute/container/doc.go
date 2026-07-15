// Package container provides a unified interface for container orchestration.
//
// Shipping backends:
//   - Memory: in-memory runtime for tests
//   - Kubernetes: client-go pod adapter
//   - Fargate: AWS ECS/Fargate adapter
//
// Reserved (not implemented — driver constants exist for future adapters):
//   - Docker Engine, raw ECS/GKE/AKS drivers (use k8s with the right kubeconfig
//     for GKE/AKS)
package container

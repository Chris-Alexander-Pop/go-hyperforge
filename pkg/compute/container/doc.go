// Package container provides a unified interface for container orchestration.
//
// Shipping backends:
//   - Memory: in-memory runtime for tests
//   - Docker: Engine API client (adapters/docker)
//   - Kubernetes: client-go pod adapter (SPDY Exec; Stats → Unimplemented without metrics-server)
//   - Fargate: AWS ECS/Fargate adapter
//
// Reserved driver names:
//   - raw ECS/GKE/AKS (use fargate or k8s with the right kubeconfig)
package container

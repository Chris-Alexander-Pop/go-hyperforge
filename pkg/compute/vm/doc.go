// Package vm provides a unified interface for virtual machine management.
//
// Shipping backends:
//   - Memory: in-memory manager for tests
//   - EC2: AWS SDK v2 (adapters/ec2) with injectable EC2API for tests
//   - GCE: google.golang.org/api/compute/v1 (adapters/gce)
//
// Scaffold:
//   - Azure VM (adapters/azurevm) — interface-compliant Unimplemented
//
// For private-cloud hypervisor lifecycle, see pkg/cloud/hypervisor.
package vm

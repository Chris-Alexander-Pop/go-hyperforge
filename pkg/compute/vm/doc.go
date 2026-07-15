// Package vm provides a unified interface for virtual machine management.
//
// Shipping backends:
//   - Memory: in-memory manager for tests
//
// Reserved (not implemented — driver constants exist for future adapters):
//   - EC2, GCE, Azure VM
//
// For private-cloud hypervisor lifecycle, see pkg/cloud/hypervisor.
package vm

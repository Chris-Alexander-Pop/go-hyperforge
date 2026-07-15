// Package firecracker provides a Firecracker microVM adapter via the HTTP API
// over a Unix domain socket (or TCP for tests).
//
// Implements hypervisor.Hypervisor Create/Start/Stop (plus Delete/Get/List against
// an in-process registry keyed by VM ID → socket actions).
package firecracker

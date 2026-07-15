// Package gce provides a Google Compute Engine adapter for vm.VMManager.
//
// Uses google.golang.org/api/compute/v1. Hard operations that need long-running
// operation polling are implemented; inject InstancesAPI for unit tests.
package gce

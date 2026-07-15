// Package libvirt provides a pure-Go remote libvirt adapter for hypervisor.Hypervisor.
//
// This avoids CGO (libvirt-go). It speaks a small JSON-over-HTTP protocol against a
// remote libvirt gateway. Set Config.BaseURL to the gateway endpoint; for tests,
// inject an http.Client pointed at httptest.
//
// Protocol (POST {BaseURL}/v1/vms ...):
//   - POST /v1/vms           CreateVM
//   - POST /v1/vms/{id}/start
//   - POST /v1/vms/{id}/stop
//   - DELETE /v1/vms/{id}
//   - GET /v1/vms/{id}
//   - GET /v1/vms
package libvirt

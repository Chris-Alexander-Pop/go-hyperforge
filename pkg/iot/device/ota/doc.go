// Package ota provides Over-the-Air update functionality for IoT devices.
//
// Downloads manifests and payloads over HTTP with SHA-256 verification and
// semantic version comparison (via pkg/iot / golang.org/x/mod/semver). Download
// retries use pkg/resilience. ApplyUpdate is a platform stub — wire install/reboot
// in the integrator. Prefer pkg/iot.Updater and adapters/memory for tests.
//
// Usage:
//
//	updater := ota.New(ota.Config{StorageURL: "https://updates.example.com"})
//	err := updater.CheckAndApply(ctx, "device-123", "1.0.0")
package ota

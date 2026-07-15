/*
Package iot defines MQTT and OTA interfaces for IoT device messaging and updates.

# Scope (honest)

This root package provides:

  - Client — MQTT publish/subscribe interface
  - Updater — Over-the-Air firmware update interface
  - Shared message/manifest types, errors, and instrumented wrappers
  - In-memory adapters under adapters/memory for tests and local use

Concrete implementations:

  - protocols/mqtt — Eclipse Paho MQTT client (MQTT 3.1.1/5.0 over TCP/TLS)
  - protocols/coap — CoAP stub client (in-process memory; UDP/DTLS not on the wire yet)
  - device/ota — HTTP manifest/download updater with SHA-256 verification
  - device/registry — DeviceRegistry interface + adapters/memory
  - adapters/awsiot — AWS IoT Core SDK wrapper (not yet behind the root Client interface)
  - adapters/greengrass — AWS Greengrass V2 SDK wrapper (management API, not MQTT)

Certificate provisioning and real CoAP UDP transport are not implemented.
AWS adapters remain SDK-coupled; prefer root interfaces + memory for new code.

# Usage

	import (
		"github.com/chris-alexander-pop/go-hyperforge/pkg/iot"
		"github.com/chris-alexander-pop/go-hyperforge/pkg/iot/adapters/memory"
	)

	client := memory.NewClient()
	_ = client.Connect(ctx)
	_ = client.Subscribe(ctx, "sensors/#", func(msg *iot.Message) { ... })

	updater := memory.NewUpdater(memory.UpdaterConfig{})
	_ = updater.CheckAndApply(ctx, "device-1", "1.0.0")
*/
package iot

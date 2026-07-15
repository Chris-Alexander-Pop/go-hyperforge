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
  - device/cert — device certificate helper types + memory CertificateProvider
  - adapters/awsiot — AWS IoT Core SDK + NewAdapter behind root Client
  - adapters/greengrass — Greengrass V2 management + NewAdapter behind root Client

Real CoAP UDP transport and cloud certificate SDK wiring remain open.
*/
package iot

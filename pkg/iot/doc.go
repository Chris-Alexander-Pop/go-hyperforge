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
  - adapters/mqtt — Paho behind root iot.Client
  - protocols/coap — CoAP Memory stub + UDP datagram listen/exchange
  - device/ota — HTTP manifest/download updater with SHA-256 verification
  - device/registry — DeviceRegistry interface + adapters/memory
  - device/cert — CertificateProvider + memory + adapters/awsiot (injectable SDK)
  - adapters/awsiot — AWS IoT Core SDK + NewAdapter behind root Client
  - adapters/greengrass — Greengrass V2 management + NewAdapter behind root Client

DTLS CoAP and Observe remain open.
*/
package iot

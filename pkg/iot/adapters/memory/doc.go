// Package memory provides in-memory MQTT and OTA adapters for testing and local development.
//
// These adapters implement pkg/iot.Client and pkg/iot.Updater with no external brokers
// or HTTP storage. They use pkg/concurrency.SmartRWMutex for observability-friendly locking.
package memory

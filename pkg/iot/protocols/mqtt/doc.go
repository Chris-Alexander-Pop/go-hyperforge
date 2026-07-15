// Package mqtt provides an MQTT client for IoT messaging via Eclipse Paho.
//
// Supports MQTT 3.1.1 and 5.0 over TCP/TLS. Token wait timeouts are handled
// correctly (timeout ≠ success). Prefer pkg/iot.Client via adapters/mqtt (or
// adapters/memory in tests) for interface-based consumers.
//
// Usage:
//
//	client, err := mqtt.New(mqtt.Config{Broker: "tcp://localhost:1883"})
//	err = client.Connect(ctx)
//	err = client.Publish(ctx, "sensors/temp", []byte("25.5"))
package mqtt

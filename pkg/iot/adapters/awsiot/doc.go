// Package awsiot provides an AWS IoT Core client and a pkg/iot.Client adapter.
//
// Use New / Client for control-plane (things/shadows) and data-plane Publish.
// Use NewAdapter / NewAdapterFromClient to expose MQTT-shaped iot.Client:
// Publish forwards to AWS; Subscribe is in-process fan-out for tests/bridges.
package awsiot

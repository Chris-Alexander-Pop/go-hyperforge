// Package awsiot provides an AWS IoT Core client (SDK wrapper).
//
// This adapter talks to the AWS IoT control and data-plane APIs. It is not yet
// wired behind pkg/iot.Client; use protocols/mqtt or adapters/memory for the
// root MQTT interface.
package awsiot

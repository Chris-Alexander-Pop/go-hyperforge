// Package gcppubsub provides a messaging.Broker backed by Google Cloud Pub/Sub.
//
// Prefer this adapter over any streaming package for Pub/Sub: pkg/streaming is
// limited to Kinesis/Event Hubs-style PutRecord producers.
package gcppubsub

// Package kafka provides a Sarama-backed messaging.Broker for Apache Kafka.
//
// Supports producer/consumer groups, partitioning, and SASL/TLS via Config.
// Construct with kafka.New — NewFromConfig in the root messaging package only
// builds the memory driver to keep SDK dependencies opt-in.
//
// Requires: github.com/IBM/sarama
package kafka

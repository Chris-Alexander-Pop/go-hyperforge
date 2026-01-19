/*
Package messaging provides a unified abstraction layer for message brokers.

This package defines the core interfaces for producing and consuming messages
across different messaging systems (Kafka, RabbitMQ, NATS, AWS SQS/SNS, GCP Pub/Sub,
Azure Service Bus).

# Architecture

The package follows the adapter pattern with decoupled dependencies:
  - Core interfaces are defined here (zero external dependencies)
  - Each adapter lives in its own sub-package (pkg/messaging/adapters/{driver})
  - Users import only the adapter they need, pulling only that SDK

# Usage

	import (
	    "github.com/chris-alexander-pop/system-design-library/pkg/messaging"
	    "github.com/chris-alexander-pop/system-design-library/pkg/messaging/adapters/kafka"
	)

	// Create a Kafka broker
	broker, err := kafka.New(kafka.Config{Brokers: []string{"localhost:9092"}})

	// Create a producer
	producer, err := broker.Producer("my-topic")
	defer producer.Close()

	// Publish a message
	err = producer.Publish(ctx, &messaging.Message{
	    ID:      uuid.New().String(),
	    Topic:   "my-topic",
	    Payload: []byte(`{"event": "user.created"}`),
	})
*/
package messaging

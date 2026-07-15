/*
Package messaging provides a unified abstraction layer for message brokers.

This package defines the core interfaces for producing and consuming messages
across different messaging systems (Kafka, RabbitMQ, NATS, AWS SQS/SNS, GCP Pub/Sub,
Azure Service Bus).

For append-only cloud stream producers (AWS Kinesis, Azure Event Hubs), use
pkg/streaming instead. GCP Pub/Sub belongs here (adapters/gcppubsub), not under
pkg/streaming.

# Factory

NewFromConfig builds the memory driver when adapters/memory is imported.
Production brokers are constructed via their adapter packages (see manager.go)
so dependents do not pull unused SDKs.

# Options

Use messaging.Publish / messaging.Consume helpers (or ApplyPublishOptions /
ContextWithConsumeOptions) to attach PublishOption / ConsumeOption hints without
changing the Producer/Consumer interfaces. Adapters may read them from message
headers or context.

# Architecture

The package follows the adapter pattern with decoupled dependencies:
  - Core interfaces are defined here (zero external broker SDKs)
  - Each adapter lives in its own sub-package (pkg/messaging/adapters/{driver})
  - Users import only the adapter they need, pulling only that SDK

# Usage

	import (
	    "github.com/chris-alexander-pop/system-design-library/pkg/messaging"
	    "github.com/chris-alexander-pop/system-design-library/pkg/messaging/adapters/memory"
	)

	broker, err := messaging.NewFromConfig(messaging.Config{Driver: "memory", BufferSize: 100})
	producer, err := broker.Producer("my-topic")
	_ = messaging.Publish(ctx, producer, &messaging.Message{Payload: []byte(`{}`)}, messaging.WithOrderingKey("k"))
*/
package messaging

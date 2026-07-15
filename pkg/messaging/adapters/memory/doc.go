// Package memory provides an in-memory messaging.Broker for tests and local use.
//
// Channel capacity per consumer group is Config.BufferSize (default 1000).
// Publish returns messaging.ErrQueueFull when any subscriber channel is full
// instead of silently dropping messages.
//
//	broker := memory.New(memory.Config{BufferSize: 100})
//	producer, _ := broker.Producer("my-topic")
//	consumer, _ := broker.Consumer("my-topic", "my-group")
//
// Importing this package registers the "memory" driver for messaging.NewFromConfig.
package memory

import (
	"github.com/chris-alexander-pop/go-hyperforge/pkg/messaging"
)

func init() {
	messaging.RegisterDriver("memory", func(cfg messaging.Config) (messaging.Broker, error) {
		return New(Config{BufferSize: cfg.BufferSize}), nil
	})
}

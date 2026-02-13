package kafka

import (
	"context"
	"fmt"
	"time"

	"github.com/IBM/sarama"
	"github.com/chris-alexander-pop/system-design-library/pkg/concurrency"
	"github.com/chris-alexander-pop/system-design-library/pkg/messaging"
)

// consumer is a Kafka consumer group implementation.
type consumer struct {
	broker        *Broker
	topic         string
	group         string
	consumerGroup sarama.ConsumerGroup
	mu            *concurrency.SmartMutex
	closed        bool
}

// newConsumer creates a new Kafka consumer.
func newConsumer(b *Broker, topic, group string, consumerGroup sarama.ConsumerGroup) *consumer {
	return &consumer{
		broker:        b,
		topic:         topic,
		group:         group,
		consumerGroup: consumerGroup,
		mu:            concurrency.NewSmartMutex(concurrency.MutexConfig{Name: "KafkaConsumer"}),
	}
}

func (c *consumer) Consume(ctx context.Context, handler messaging.MessageHandler) error {
	topics := []string{c.topic}

	consumerHandler := &consumerGroupHandler{
		handler: handler,
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			err := c.consumerGroup.Consume(ctx, topics, consumerHandler)
			if err != nil {
				if err == sarama.ErrClosedConsumerGroup {
					return nil
				}
				return messaging.ErrConsumeFailed(err)
			}
			// Consumer group rebalanced, continue
		}
	}
}

func (c *consumer) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}
	c.closed = true

	return c.consumerGroup.Close()
}

// consumerGroupHandler implements sarama.ConsumerGroupHandler
type consumerGroupHandler struct {
	handler messaging.MessageHandler
}

func (h *consumerGroupHandler) Setup(_ sarama.ConsumerGroupSession) error {
	return nil
}

func (h *consumerGroupHandler) Cleanup(_ sarama.ConsumerGroupSession) error {
	return nil
}

func (h *consumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for kafkaMsg := range claim.Messages() {
		msg := convertKafkaMessage(kafkaMsg)

		ctx := session.Context()
		err := h.handler(ctx, msg)
		if err != nil {
			// Don't mark as consumed on error - message will be redelivered
			continue
		}

		session.MarkMessage(kafkaMsg, "")
	}
	return nil
}

func convertKafkaMessage(kafkaMsg *sarama.ConsumerMessage) *messaging.Message {
	msg := &messaging.Message{
		Topic:     kafkaMsg.Topic,
		Key:       kafkaMsg.Key,
		Payload:   kafkaMsg.Value,
		Timestamp: kafkaMsg.Timestamp,
		Headers:   make(map[string]string),
		Metadata: messaging.MessageMetadata{
			Partition: kafkaMsg.Partition,
			Offset:    kafkaMsg.Offset,
			Raw:       kafkaMsg,
		},
	}

	// Extract headers
	for _, h := range kafkaMsg.Headers {
		key := string(h.Key)
		value := string(h.Value)
		if key == "message-id" {
			msg.ID = value
		} else {
			msg.Headers[key] = value
		}
	}

	// If no message ID was in headers, generate one from partition+offset
	if msg.ID == "" {
		msg.ID = fmt.Sprintf("%s-%d-%d", time.Now().Format("20060102150405"), kafkaMsg.Partition, kafkaMsg.Offset)
	}

	return msg
}

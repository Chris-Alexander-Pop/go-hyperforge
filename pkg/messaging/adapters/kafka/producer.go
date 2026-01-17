package kafka

import (
	"context"
	"time"

	"github.com/IBM/sarama"
	"github.com/chris-alexander-pop/system-design-library/pkg/messaging"
	"github.com/google/uuid"
)

// producer is a Kafka sync producer implementation.
type producer struct {
	broker   *Broker
	topic    string
	producer sarama.SyncProducer
}

func (p *producer) Publish(ctx context.Context, msg *messaging.Message) error {
	if msg.ID == "" {
		msg.ID = uuid.New().String()
	}
	if msg.Timestamp.IsZero() {
		msg.Timestamp = time.Now()
	}

	kafkaMsg := &sarama.ProducerMessage{
		Topic:     p.topic,
		Value:     sarama.ByteEncoder(msg.Payload),
		Timestamp: msg.Timestamp,
	}

	// Set key for partitioning if provided
	if len(msg.Key) > 0 {
		kafkaMsg.Key = sarama.ByteEncoder(msg.Key)
	}

	// Add headers
	if len(msg.Headers) > 0 {
		for k, v := range msg.Headers {
			kafkaMsg.Headers = append(kafkaMsg.Headers, sarama.RecordHeader{
				Key:   []byte(k),
				Value: []byte(v),
			})
		}
	}

	// Add message ID as header
	kafkaMsg.Headers = append(kafkaMsg.Headers, sarama.RecordHeader{
		Key:   []byte("message-id"),
		Value: []byte(msg.ID),
	})

	partition, offset, err := p.producer.SendMessage(kafkaMsg)
	if err != nil {
		return messaging.ErrPublishFailed(err)
	}

	// Update metadata
	msg.Metadata.Partition = partition
	msg.Metadata.Offset = offset

	return nil
}

func (p *producer) PublishBatch(ctx context.Context, msgs []*messaging.Message) error {
	kafkaMsgs := make([]*sarama.ProducerMessage, len(msgs))

	for i, msg := range msgs {
		if msg.ID == "" {
			msg.ID = uuid.New().String()
		}
		if msg.Timestamp.IsZero() {
			msg.Timestamp = time.Now()
		}

		kafkaMsg := &sarama.ProducerMessage{
			Topic:     p.topic,
			Value:     sarama.ByteEncoder(msg.Payload),
			Timestamp: msg.Timestamp,
		}

		if len(msg.Key) > 0 {
			kafkaMsg.Key = sarama.ByteEncoder(msg.Key)
		}

		if len(msg.Headers) > 0 {
			for k, v := range msg.Headers {
				kafkaMsg.Headers = append(kafkaMsg.Headers, sarama.RecordHeader{
					Key:   []byte(k),
					Value: []byte(v),
				})
			}
		}

		kafkaMsg.Headers = append(kafkaMsg.Headers, sarama.RecordHeader{
			Key:   []byte("message-id"),
			Value: []byte(msg.ID),
		})

		kafkaMsgs[i] = kafkaMsg
	}

	// Send all messages
	err := p.producer.SendMessages(kafkaMsgs)
	if err != nil {
		return messaging.ErrPublishFailed(err)
	}

	return nil
}

func (p *producer) Close() error {
	return p.producer.Close()
}

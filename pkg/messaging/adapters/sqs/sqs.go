// Package sqs provides an AWS SQS messaging adapter.
//
// This adapter implements the messaging.Broker interface for Amazon Simple Queue Service,
// supporting standard queues, FIFO queues, long polling, and dead letter queues.
//
// # Usage
//
//	cfg := sqs.Config{
//	    Region:   "us-east-1",
//	    QueueURL: "https://sqs.us-east-1.amazonaws.com/123456789/my-queue",
//	}
//	broker, err := sqs.New(ctx, cfg)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer broker.Close()
//
// # Features
//
//   - Standard queues: At-least-once delivery
//   - FIFO queues: Exactly-once, ordered delivery
//   - Long polling: Reduced costs with up to 20s wait time
//   - Dead letter queues: Automatic handling of failed messages
//
// # Dependencies
//
// This package requires: github.com/aws/aws-sdk-go-v2/service/sqs
package sqs

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/chris-alexander-pop/system-design-library/pkg/messaging"
	"github.com/google/uuid"
)

// Config holds configuration for the SQS broker.
type Config struct {
	// Region is the AWS region.
	Region string `env:"AWS_REGION" env-default:"us-east-1"`

	// QueueURL is the SQS queue URL. If not provided, uses QueueName to look up.
	QueueURL string `env:"SQS_QUEUE_URL"`

	// QueueName is used to look up the queue URL if QueueURL is not provided.
	QueueName string `env:"SQS_QUEUE_NAME"`

	// MaxMessages is the maximum number of messages to receive per poll (1-10).
	MaxMessages int32 `env:"SQS_MAX_MESSAGES" env-default:"10"`

	// WaitTimeSeconds enables long polling (0-20).
	WaitTimeSeconds int32 `env:"SQS_WAIT_TIME" env-default:"20"`

	// VisibilityTimeout is how long messages are hidden after receive (seconds).
	VisibilityTimeout int32 `env:"SQS_VISIBILITY_TIMEOUT" env-default:"30"`

	// FIFO enables FIFO queue behavior.
	FIFO bool `env:"SQS_FIFO" env-default:"false"`

	// Endpoint is a custom endpoint URL (for LocalStack, etc.).
	Endpoint string `env:"SQS_ENDPOINT"`

	// AccessKeyID and SecretAccessKey for explicit credentials.
	// If empty, uses default credential chain.
	AccessKeyID     string `env:"AWS_ACCESS_KEY_ID"`
	SecretAccessKey string `env:"AWS_SECRET_ACCESS_KEY"`
}

// Broker is an SQS message broker implementation.
type Broker struct {
	config   Config
	client   *sqs.Client
	queueURL string
	mu       sync.RWMutex
	closed   bool
}

// New creates a new SQS broker.
func New(ctx context.Context, cfg Config) (*Broker, error) {
	opts := []func(*config.LoadOptions) error{
		config.WithRegion(cfg.Region),
	}

	// Explicit credentials if provided
	if cfg.AccessKeyID != "" && cfg.SecretAccessKey != "" {
		opts = append(opts, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		))
	}

	awsCfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, messaging.ErrConnectionFailed(err)
	}

	clientOpts := []func(*sqs.Options){}
	if cfg.Endpoint != "" {
		clientOpts = append(clientOpts, func(o *sqs.Options) {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
		})
	}

	client := sqs.NewFromConfig(awsCfg, clientOpts...)

	// Resolve queue URL if not provided
	queueURL := cfg.QueueURL
	if queueURL == "" && cfg.QueueName != "" {
		result, err := client.GetQueueUrl(ctx, &sqs.GetQueueUrlInput{
			QueueName: aws.String(cfg.QueueName),
		})
		if err != nil {
			return nil, messaging.ErrTopicNotFound(cfg.QueueName, err)
		}
		queueURL = *result.QueueUrl
	}

	return &Broker{
		config:   cfg,
		client:   client,
		queueURL: queueURL,
	}, nil
}

// Producer creates a producer for the SQS queue.
// Note: In SQS, the topic parameter is ignored as the queue is fixed at broker creation.
func (b *Broker) Producer(topic string) (messaging.Producer, error) {
	b.mu.RLock()
	if b.closed {
		b.mu.RUnlock()
		return nil, messaging.ErrClosed(nil)
	}
	b.mu.RUnlock()

	return &producer{
		broker: b,
	}, nil
}

// Consumer creates a consumer for the SQS queue.
// Note: The group parameter is ignored for SQS (use separate queues for different consumers).
func (b *Broker) Consumer(topic string, group string) (messaging.Consumer, error) {
	b.mu.RLock()
	if b.closed {
		b.mu.RUnlock()
		return nil, messaging.ErrClosed(nil)
	}
	b.mu.RUnlock()

	return &consumer{
		broker: b,
	}, nil
}

// Close shuts down the SQS broker.
func (b *Broker) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return nil
	}
	b.closed = true

	return nil
}

// Healthy checks if the SQS connection is healthy.
func (b *Broker) Healthy(ctx context.Context) bool {
	b.mu.RLock()
	if b.closed {
		b.mu.RUnlock()
		return false
	}
	b.mu.RUnlock()

	// Try to get queue attributes
	_, err := b.client.GetQueueAttributes(ctx, &sqs.GetQueueAttributesInput{
		QueueUrl:       aws.String(b.queueURL),
		AttributeNames: []types.QueueAttributeName{types.QueueAttributeNameQueueArn},
	})
	return err == nil
}

// producer is an SQS producer.
type producer struct {
	broker *Broker
}

func (p *producer) Publish(ctx context.Context, msg *messaging.Message) error {
	if msg.ID == "" {
		msg.ID = uuid.New().String()
	}
	if msg.Timestamp.IsZero() {
		msg.Timestamp = time.Now()
	}

	input := &sqs.SendMessageInput{
		QueueUrl:    aws.String(p.broker.queueURL),
		MessageBody: aws.String(string(msg.Payload)),
		MessageAttributes: map[string]types.MessageAttributeValue{
			"MessageId": {
				DataType:    aws.String("String"),
				StringValue: aws.String(msg.ID),
			},
			"Timestamp": {
				DataType:    aws.String("Number"),
				StringValue: aws.String(strconv.FormatInt(msg.Timestamp.UnixMilli(), 10)),
			},
		},
	}

	// Add custom headers as message attributes
	for k, v := range msg.Headers {
		input.MessageAttributes[k] = types.MessageAttributeValue{
			DataType:    aws.String("String"),
			StringValue: aws.String(v),
		}
	}

	// FIFO-specific attributes
	if p.broker.config.FIFO {
		// Use key as message group ID if provided
		if len(msg.Key) > 0 {
			input.MessageGroupId = aws.String(string(msg.Key))
		} else {
			input.MessageGroupId = aws.String("default")
		}
		input.MessageDeduplicationId = aws.String(msg.ID)
	}

	result, err := p.broker.client.SendMessage(ctx, input)
	if err != nil {
		return messaging.ErrPublishFailed(err)
	}

	if result.MessageId != nil {
		msg.Metadata.ReceiptHandle = *result.MessageId
	}

	return nil
}

func (p *producer) PublishBatch(ctx context.Context, msgs []*messaging.Message) error {
	entries := make([]types.SendMessageBatchRequestEntry, len(msgs))

	for i, msg := range msgs {
		if msg.ID == "" {
			msg.ID = uuid.New().String()
		}
		if msg.Timestamp.IsZero() {
			msg.Timestamp = time.Now()
		}

		entry := types.SendMessageBatchRequestEntry{
			Id:          aws.String(strconv.Itoa(i)),
			MessageBody: aws.String(string(msg.Payload)),
			MessageAttributes: map[string]types.MessageAttributeValue{
				"MessageId": {
					DataType:    aws.String("String"),
					StringValue: aws.String(msg.ID),
				},
			},
		}

		if p.broker.config.FIFO {
			if len(msg.Key) > 0 {
				entry.MessageGroupId = aws.String(string(msg.Key))
			} else {
				entry.MessageGroupId = aws.String("default")
			}
			entry.MessageDeduplicationId = aws.String(msg.ID)
		}

		entries[i] = entry
	}

	_, err := p.broker.client.SendMessageBatch(ctx, &sqs.SendMessageBatchInput{
		QueueUrl: aws.String(p.broker.queueURL),
		Entries:  entries,
	})
	if err != nil {
		return messaging.ErrPublishFailed(err)
	}

	return nil
}

func (p *producer) Close() error {
	return nil
}

// consumer is an SQS consumer.
type consumer struct {
	broker *Broker
	mu     sync.Mutex
	closed bool
}

func (c *consumer) Consume(ctx context.Context, handler messaging.MessageHandler) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		c.mu.Lock()
		if c.closed {
			c.mu.Unlock()
			return nil
		}
		c.mu.Unlock()

		result, err := c.broker.client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
			QueueUrl:              aws.String(c.broker.queueURL),
			MaxNumberOfMessages:   c.broker.config.MaxMessages,
			WaitTimeSeconds:       c.broker.config.WaitTimeSeconds,
			VisibilityTimeout:     c.broker.config.VisibilityTimeout,
			MessageAttributeNames: []string{"All"},
			AttributeNames:        []types.QueueAttributeName{types.QueueAttributeNameAll},
		})
		if err != nil {
			continue // Retry on error
		}

		for _, sqsMsg := range result.Messages {
			msg := convertSQSMessage(sqsMsg)

			err := handler(ctx, msg)
			if err != nil {
				// Change visibility to allow reprocessing
				c.broker.client.ChangeMessageVisibility(ctx, &sqs.ChangeMessageVisibilityInput{
					QueueUrl:          aws.String(c.broker.queueURL),
					ReceiptHandle:     sqsMsg.ReceiptHandle,
					VisibilityTimeout: 0, // Immediately visible
				})
				continue
			}

			// Delete message on success
			c.broker.client.DeleteMessage(ctx, &sqs.DeleteMessageInput{
				QueueUrl:      aws.String(c.broker.queueURL),
				ReceiptHandle: sqsMsg.ReceiptHandle,
			})
		}
	}
}

func (c *consumer) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.closed = true
	return nil
}

func convertSQSMessage(sqsMsg types.Message) *messaging.Message {
	msg := &messaging.Message{
		Payload:   []byte(aws.ToString(sqsMsg.Body)),
		Headers:   make(map[string]string),
		Timestamp: time.Now(),
		Metadata: messaging.MessageMetadata{
			ReceiptHandle: aws.ToString(sqsMsg.ReceiptHandle),
			Raw:           sqsMsg,
		},
	}

	// Extract message ID from attributes
	if attr, ok := sqsMsg.MessageAttributes["MessageId"]; ok {
		msg.ID = aws.ToString(attr.StringValue)
	}
	if msg.ID == "" && sqsMsg.MessageId != nil {
		msg.ID = *sqsMsg.MessageId
	}

	// Extract timestamp
	if attr, ok := sqsMsg.MessageAttributes["Timestamp"]; ok {
		if ts, err := strconv.ParseInt(aws.ToString(attr.StringValue), 10, 64); err == nil {
			msg.Timestamp = time.UnixMilli(ts)
		}
	}

	// Extract custom headers
	for k, v := range sqsMsg.MessageAttributes {
		if k != "MessageId" && k != "Timestamp" {
			msg.Headers[k] = aws.ToString(v.StringValue)
		}
	}

	// Approximate receive count
	if countStr, ok := sqsMsg.Attributes["ApproximateReceiveCount"]; ok {
		if count, err := strconv.Atoi(countStr); err == nil {
			msg.Metadata.DeliveryCount = count
		}
	}

	return msg
}

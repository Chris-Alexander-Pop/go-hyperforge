// Package sns provides an AWS SNS messaging adapter.
//
// This adapter implements a publish-only messaging.Broker for Amazon Simple Notification Service.
// SNS is a fan-out pub/sub service - for consuming messages, subscribe an SQS queue to the SNS topic.
//
// # Usage
//
//	cfg := sns.Config{
//	    Region:   "us-east-1",
//	    TopicARN: "arn:aws:sns:us-east-1:123456789:my-topic",
//	}
//	broker, err := sns.New(ctx, cfg)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer broker.Close()
//
// # Consuming SNS Messages
//
// SNS doesn't directly support message consumption. To consume:
//  1. Create an SQS queue
//  2. Subscribe the queue to your SNS topic
//  3. Use the SQS adapter to consume messages
//
// # Dependencies
//
// This package requires: github.com/aws/aws-sdk-go-v2/service/sns
package sns

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sns/types"
	"github.com/chris-alexander-pop/system-design-library/pkg/messaging"
	"github.com/google/uuid"
)

// Config holds configuration for the SNS broker.
type Config struct {
	// Region is the AWS region.
	Region string `env:"AWS_REGION" env-default:"us-east-1"`

	// TopicARN is the SNS topic ARN.
	TopicARN string `env:"SNS_TOPIC_ARN"`

	// TopicName is used to create or look up the topic if TopicARN is not provided.
	TopicName string `env:"SNS_TOPIC_NAME"`

	// FIFO enables FIFO topic behavior.
	FIFO bool `env:"SNS_FIFO" env-default:"false"`

	// Endpoint is a custom endpoint URL (for LocalStack, etc.).
	Endpoint string `env:"SNS_ENDPOINT"`

	// AccessKeyID and SecretAccessKey for explicit credentials.
	AccessKeyID     string `env:"AWS_ACCESS_KEY_ID"`
	SecretAccessKey string `env:"AWS_SECRET_ACCESS_KEY"`
}

// Broker is an SNS message broker implementation.
// Note: SNS only supports publishing. For consuming, use SQS subscribed to SNS.
type Broker struct {
	config   Config
	client   *sns.Client
	topicARN string
	mu       sync.RWMutex
	closed   bool
}

// New creates a new SNS broker.
func New(ctx context.Context, cfg Config) (*Broker, error) {
	opts := []func(*config.LoadOptions) error{
		config.WithRegion(cfg.Region),
	}

	if cfg.AccessKeyID != "" && cfg.SecretAccessKey != "" {
		opts = append(opts, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		))
	}

	awsCfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, messaging.ErrConnectionFailed(err)
	}

	clientOpts := []func(*sns.Options){}
	if cfg.Endpoint != "" {
		clientOpts = append(clientOpts, func(o *sns.Options) {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
		})
	}

	client := sns.NewFromConfig(awsCfg, clientOpts...)

	topicARN := cfg.TopicARN

	// Create topic if name provided but not ARN
	if topicARN == "" && cfg.TopicName != "" {
		input := &sns.CreateTopicInput{
			Name: aws.String(cfg.TopicName),
		}

		if cfg.FIFO {
			input.Attributes = map[string]string{
				"FifoTopic": "true",
			}
		}

		result, err := client.CreateTopic(ctx, input)
		if err != nil {
			return nil, messaging.ErrConnectionFailed(err)
		}
		topicARN = *result.TopicArn
	}

	return &Broker{
		config:   cfg,
		client:   client,
		topicARN: topicARN,
	}, nil
}

// Producer creates a producer for the SNS topic.
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

// Consumer is not supported for SNS - use SQS subscribed to the SNS topic.
func (b *Broker) Consumer(topic string, group string) (messaging.Consumer, error) {
	return nil, messaging.ErrInvalidConfig("SNS does not support direct consumption. Subscribe an SQS queue to this topic and use the SQS adapter.", nil)
}

// Close shuts down the SNS broker.
func (b *Broker) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return nil
	}
	b.closed = true

	return nil
}

// Healthy checks if the SNS connection is healthy.
func (b *Broker) Healthy(ctx context.Context) bool {
	b.mu.RLock()
	if b.closed {
		b.mu.RUnlock()
		return false
	}
	b.mu.RUnlock()

	_, err := b.client.GetTopicAttributes(ctx, &sns.GetTopicAttributesInput{
		TopicArn: aws.String(b.topicARN),
	})
	return err == nil
}

// producer is an SNS producer.
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

	input := &sns.PublishInput{
		TopicArn: aws.String(p.broker.topicARN),
		Message:  aws.String(string(msg.Payload)),
		MessageAttributes: map[string]types.MessageAttributeValue{
			"MessageId": {
				DataType:    aws.String("String"),
				StringValue: aws.String(msg.ID),
			},
		},
	}

	// Add headers as message attributes
	for k, v := range msg.Headers {
		input.MessageAttributes[k] = types.MessageAttributeValue{
			DataType:    aws.String("String"),
			StringValue: aws.String(v),
		}
	}

	// FIFO-specific attributes
	if p.broker.config.FIFO {
		if len(msg.Key) > 0 {
			input.MessageGroupId = aws.String(string(msg.Key))
		} else {
			input.MessageGroupId = aws.String("default")
		}
		input.MessageDeduplicationId = aws.String(msg.ID)
	}

	// Add subject if topic is set in message (used for filtering)
	if msg.Topic != "" {
		input.Subject = aws.String(msg.Topic)
	}

	result, err := p.broker.client.Publish(ctx, input)
	if err != nil {
		return messaging.ErrPublishFailed(err)
	}

	if result.MessageId != nil {
		msg.Metadata.ReceiptHandle = *result.MessageId
	}

	return nil
}

func (p *producer) PublishBatch(ctx context.Context, msgs []*messaging.Message) error {
	entries := make([]types.PublishBatchRequestEntry, len(msgs))

	for i, msg := range msgs {
		if msg.ID == "" {
			msg.ID = uuid.New().String()
		}
		if msg.Timestamp.IsZero() {
			msg.Timestamp = time.Now()
		}

		entry := types.PublishBatchRequestEntry{
			Id:      aws.String(msg.ID),
			Message: aws.String(string(msg.Payload)),
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

	_, err := p.broker.client.PublishBatch(ctx, &sns.PublishBatchInput{
		TopicArn:                   aws.String(p.broker.topicARN),
		PublishBatchRequestEntries: entries,
	})
	if err != nil {
		return messaging.ErrPublishFailed(err)
	}

	return nil
}

func (p *producer) Close() error {
	return nil
}

// SNSMessageWrapper represents an SNS message when delivered to SQS.
// Use this to unmarshal SNS messages received via SQS subscription.
type SNSMessageWrapper struct {
	Type              string `json:"Type"`
	MessageId         string `json:"MessageId"`
	TopicArn          string `json:"TopicArn"`
	Subject           string `json:"Subject,omitempty"`
	Message           string `json:"Message"`
	Timestamp         string `json:"Timestamp"`
	SignatureVersion  string `json:"SignatureVersion"`
	Signature         string `json:"Signature"`
	SigningCertURL    string `json:"SigningCertURL"`
	UnsubscribeURL    string `json:"UnsubscribeURL"`
	MessageAttributes map[string]struct {
		Type  string `json:"Type"`
		Value string `json:"Value"`
	} `json:"MessageAttributes,omitempty"`
}

// ParseSNSMessage extracts the original message from an SNS wrapper.
// Use this when consuming SNS messages via SQS.
func ParseSNSMessage(payload []byte) (*messaging.Message, error) {
	var wrapper SNSMessageWrapper
	if err := json.Unmarshal(payload, &wrapper); err != nil {
		return nil, err
	}

	msg := &messaging.Message{
		ID:      wrapper.MessageId,
		Topic:   wrapper.Subject,
		Payload: []byte(wrapper.Message),
		Headers: make(map[string]string),
	}

	if wrapper.Timestamp != "" {
		if t, err := time.Parse(time.RFC3339Nano, wrapper.Timestamp); err == nil {
			msg.Timestamp = t
		}
	}

	for k, v := range wrapper.MessageAttributes {
		msg.Headers[k] = v.Value
	}

	return msg, nil
}

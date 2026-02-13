// Package kafka provides a Kafka messaging adapter using Sarama.
//
// This adapter implements the messaging.Broker interface for Apache Kafka,
// supporting producer/consumer groups, partitioning, and SASL/TLS authentication.
//
// # Usage
//
//	cfg := kafka.Config{
//	    Brokers: []string{"localhost:9092"},
//	    Version: "3.6.0",
//	}
//	broker, err := kafka.New(cfg)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer broker.Close()
//
// # Dependencies
//
// This package requires: github.com/IBM/sarama
package kafka

import (
	"context"
	"crypto/tls"
	"strings"
	"time"

	"github.com/IBM/sarama"
	"github.com/chris-alexander-pop/system-design-library/pkg/concurrency"
	"github.com/chris-alexander-pop/system-design-library/pkg/messaging"
)

// Config holds configuration for the Kafka broker.
type Config struct {
	// Brokers is a list of Kafka broker addresses.
	Brokers []string `env:"KAFKA_BROKERS" env-default:"localhost:9092"`

	// Version is the Kafka version string (e.g., "3.6.0").
	Version string `env:"KAFKA_VERSION" env-default:"3.6.0"`

	// ClientID identifies this client to the broker.
	ClientID string `env:"KAFKA_CLIENT_ID" env-default:"system-design-library"`

	// SASL configuration for authentication.
	SASL *SASLConfig

	// TLS configuration for encryption.
	TLS *TLSConfig

	// Producer settings
	ProducerRequiredAcks string        `env:"KAFKA_PRODUCER_ACKS" env-default:"all"` // none, local, all
	ProducerRetryMax     int           `env:"KAFKA_PRODUCER_RETRY_MAX" env-default:"3"`
	ProducerFlushFreq    time.Duration `env:"KAFKA_PRODUCER_FLUSH_FREQ" env-default:"100ms"`

	// Consumer settings
	ConsumerGroup          string        `env:"KAFKA_CONSUMER_GROUP" env-default:"default-group"`
	ConsumerOffsetsInitial int64         `env:"KAFKA_CONSUMER_OFFSETS_INITIAL" env-default:"-1"` // -1 = newest, -2 = oldest
	ConsumerSessionTimeout time.Duration `env:"KAFKA_CONSUMER_SESSION_TIMEOUT" env-default:"10s"`
	ConsumerHeartbeat      time.Duration `env:"KAFKA_CONSUMER_HEARTBEAT" env-default:"3s"`
}

// SASLConfig holds SASL authentication configuration.
type SASLConfig struct {
	Enable    bool   `env:"KAFKA_SASL_ENABLE" env-default:"false"`
	Mechanism string `env:"KAFKA_SASL_MECHANISM" env-default:"PLAIN"` // PLAIN, SCRAM-SHA-256, SCRAM-SHA-512
	Username  string `env:"KAFKA_SASL_USERNAME"`
	Password  string `env:"KAFKA_SASL_PASSWORD"`
}

// TLSConfig holds TLS configuration.
type TLSConfig struct {
	Enable             bool   `env:"KAFKA_TLS_ENABLE" env-default:"false"`
	InsecureSkipVerify bool   `env:"KAFKA_TLS_INSECURE" env-default:"false"`
	CertFile           string `env:"KAFKA_TLS_CERT_FILE"`
	KeyFile            string `env:"KAFKA_TLS_KEY_FILE"`
	CAFile             string `env:"KAFKA_TLS_CA_FILE"`
}

// Broker is a Kafka message broker implementation.
type Broker struct {
	config       Config
	saramaConfig *sarama.Config
	client       sarama.Client
	mu           *concurrency.SmartRWMutex
	closed       bool
}

// New creates a new Kafka broker.
func New(cfg Config) (*Broker, error) {
	saramaCfg := buildSaramaConfig(cfg)

	// Parse brokers if provided as comma-separated string
	brokers := cfg.Brokers
	if len(brokers) == 1 && strings.Contains(brokers[0], ",") {
		brokers = strings.Split(brokers[0], ",")
	}

	client, err := sarama.NewClient(brokers, saramaCfg)
	if err != nil {
		return nil, messaging.ErrConnectionFailed(err)
	}

	return &Broker{
		config:       cfg,
		saramaConfig: saramaCfg,
		client:       client,
		mu:           concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "KafkaBroker"}),
	}, nil
}

func buildSaramaConfig(cfg Config) *sarama.Config {
	saramaCfg := sarama.NewConfig()

	// Version
	version, err := sarama.ParseKafkaVersion(cfg.Version)
	if err != nil {
		version = sarama.V3_6_0_0
	}
	saramaCfg.Version = version

	// Client ID
	if cfg.ClientID != "" {
		saramaCfg.ClientID = cfg.ClientID
	}

	// Producer settings
	saramaCfg.Producer.Return.Successes = true
	saramaCfg.Producer.Return.Errors = true
	saramaCfg.Producer.Retry.Max = cfg.ProducerRetryMax
	if cfg.ProducerFlushFreq > 0 {
		saramaCfg.Producer.Flush.Frequency = cfg.ProducerFlushFreq
	}

	switch cfg.ProducerRequiredAcks {
	case "none":
		saramaCfg.Producer.RequiredAcks = sarama.NoResponse
	case "local":
		saramaCfg.Producer.RequiredAcks = sarama.WaitForLocal
	default:
		saramaCfg.Producer.RequiredAcks = sarama.WaitForAll
	}

	// Consumer settings
	saramaCfg.Consumer.Offsets.Initial = cfg.ConsumerOffsetsInitial
	if cfg.ConsumerSessionTimeout > 0 {
		saramaCfg.Consumer.Group.Session.Timeout = cfg.ConsumerSessionTimeout
	}
	if cfg.ConsumerHeartbeat > 0 {
		saramaCfg.Consumer.Group.Heartbeat.Interval = cfg.ConsumerHeartbeat
	}

	// SASL
	if cfg.SASL != nil && cfg.SASL.Enable {
		saramaCfg.Net.SASL.Enable = true
		saramaCfg.Net.SASL.User = cfg.SASL.Username
		saramaCfg.Net.SASL.Password = cfg.SASL.Password

		switch cfg.SASL.Mechanism {
		case "SCRAM-SHA-256":
			saramaCfg.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA256
			saramaCfg.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient {
				return &scramClient{HashGeneratorFcn: SHA256}
			}
		case "SCRAM-SHA-512":
			saramaCfg.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA512
			saramaCfg.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient {
				return &scramClient{HashGeneratorFcn: SHA512}
			}
		default:
			saramaCfg.Net.SASL.Mechanism = sarama.SASLTypePlaintext
		}
	}

	// TLS
	if cfg.TLS != nil && cfg.TLS.Enable {
		saramaCfg.Net.TLS.Enable = true
		saramaCfg.Net.TLS.Config = &tls.Config{
			InsecureSkipVerify: cfg.TLS.InsecureSkipVerify,
		}
	}

	return saramaCfg
}

// Producer creates a new Kafka producer for the specified topic.
func (b *Broker) Producer(topic string) (messaging.Producer, error) {
	b.mu.RLock()
	if b.closed {
		b.mu.RUnlock()
		return nil, messaging.ErrClosed(nil)
	}
	b.mu.RUnlock()

	syncProducer, err := sarama.NewSyncProducerFromClient(b.client)
	if err != nil {
		return nil, messaging.ErrConnectionFailed(err)
	}

	return &producer{
		broker:   b,
		topic:    topic,
		producer: syncProducer,
	}, nil
}

// Consumer creates a new Kafka consumer group for the specified topic.
func (b *Broker) Consumer(topic string, group string) (messaging.Consumer, error) {
	b.mu.RLock()
	if b.closed {
		b.mu.RUnlock()
		return nil, messaging.ErrClosed(nil)
	}
	b.mu.RUnlock()

	if group == "" {
		group = b.config.ConsumerGroup
	}

	consumerGroup, err := sarama.NewConsumerGroupFromClient(group, b.client)
	if err != nil {
		return nil, messaging.ErrConnectionFailed(err)
	}

	return newConsumer(b, topic, group, consumerGroup), nil
}

// Close shuts down the Kafka broker connection.
func (b *Broker) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return nil
	}
	b.closed = true

	return b.client.Close()
}

// Healthy returns true if the broker connection is healthy.
func (b *Broker) Healthy(ctx context.Context) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.closed {
		return false
	}

	// Try to get controller - if this works, we're connected
	_, err := b.client.Controller()
	return err == nil
}

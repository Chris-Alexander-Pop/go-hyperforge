package mqtt

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"os"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/concurrency"
	pkgerrors "github.com/chris-alexander-pop/system-design-library/pkg/errors"
	paho "github.com/eclipse/paho.mqtt.golang"
)

// QoS levels for MQTT messages.
type QoS byte

const (
	QoSAtMostOnce  QoS = 0 // Fire and forget
	QoSAtLeastOnce QoS = 1 // Acknowledged delivery
	QoSExactlyOnce QoS = 2 // Ensured delivery
)

// Config holds MQTT client configuration.
type Config struct {
	// Broker is the MQTT broker URL (tcp://host:port or ssl://host:port)
	Broker string

	// ClientID is the unique client identifier
	ClientID string

	// Username for authentication
	Username string

	// Password for authentication
	Password string

	// CleanSession starts a clean session
	CleanSession bool

	// KeepAlive interval in seconds
	KeepAlive time.Duration

	// ConnectTimeout for connection attempts
	ConnectTimeout time.Duration

	// TLS configuration
	TLSConfig *TLSConfig

	// AutoReconnect enables automatic reconnection
	AutoReconnect bool

	// MaxReconnectInterval is the max time between reconnect attempts
	MaxReconnectInterval time.Duration
}

// TLSConfig holds TLS settings.
type TLSConfig struct {
	// CAFile is the CA certificate file
	CAFile string

	// CertFile is the client certificate file
	CertFile string

	// KeyFile is the client key file
	KeyFile string

	// InsecureSkipVerify skips certificate verification
	InsecureSkipVerify bool
}

// Message represents an MQTT message.
type Message struct {
	Topic     string
	Payload   []byte
	QoS       QoS
	Retained  bool
	MessageID uint16
}

// MessageHandler is called when a message is received.
type MessageHandler func(msg *Message)

// Client provides MQTT operations via Eclipse Paho.
// Prefer pkg/iot.Client for interface-based consumers; use adapters/memory in tests.
type Client struct {
	client   paho.Client
	config   Config
	handlers map[string]MessageHandler
	mu       *concurrency.SmartRWMutex
}

// New creates a new MQTT client.
func New(cfg Config) (*Client, error) {
	if cfg.Broker == "" {
		return nil, pkgerrors.InvalidArgument("broker URL required", nil)
	}
	if cfg.ClientID == "" {
		cfg.ClientID = "mqtt-client-" + time.Now().Format("20060102150405")
	}
	if cfg.KeepAlive == 0 {
		cfg.KeepAlive = 60 * time.Second
	}
	if cfg.ConnectTimeout == 0 {
		cfg.ConnectTimeout = 30 * time.Second
	}
	if cfg.MaxReconnectInterval == 0 {
		cfg.MaxReconnectInterval = 10 * time.Minute
	}

	opts := paho.NewClientOptions().
		AddBroker(cfg.Broker).
		SetClientID(cfg.ClientID).
		SetCleanSession(cfg.CleanSession).
		SetKeepAlive(cfg.KeepAlive).
		SetConnectTimeout(cfg.ConnectTimeout).
		SetAutoReconnect(cfg.AutoReconnect).
		SetMaxReconnectInterval(cfg.MaxReconnectInterval)

	if cfg.Username != "" {
		opts.SetUsername(cfg.Username)
		opts.SetPassword(cfg.Password)
	}

	if cfg.TLSConfig != nil {
		tlsConfig, err := newTLSConfig(cfg.TLSConfig)
		if err != nil {
			return nil, err
		}
		opts.SetTLSConfig(tlsConfig)
	}

	c := &Client{
		config:   cfg,
		handlers: make(map[string]MessageHandler),
		mu:       concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "mqtt-client"}),
	}

	opts.SetDefaultPublishHandler(c.defaultHandler)
	opts.SetOnConnectHandler(c.onConnect)

	c.client = paho.NewClient(opts)

	return c, nil
}

func newTLSConfig(cfg *TLSConfig) (*tls.Config, error) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: cfg.InsecureSkipVerify,
	}

	if cfg.CAFile != "" {
		caCert, err := os.ReadFile(cfg.CAFile)
		if err != nil {
			return nil, pkgerrors.Internal("failed to read CA file", err)
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)
		tlsConfig.RootCAs = caCertPool
	}

	if cfg.CertFile != "" && cfg.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
		if err != nil {
			return nil, pkgerrors.Internal("failed to load client certificate", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	return tlsConfig, nil
}

func (c *Client) defaultHandler(client paho.Client, msg paho.Message) {
	c.mu.RLock()
	handler, ok := c.handlers[msg.Topic()]
	c.mu.RUnlock()

	if ok {
		handler(&Message{
			Topic:     msg.Topic(),
			Payload:   msg.Payload(),
			QoS:       QoS(msg.Qos()),
			Retained:  msg.Retained(),
			MessageID: msg.MessageID(),
		})
	}
}

func (c *Client) onConnect(client paho.Client) {
	// Re-subscribe to all topics on reconnect
	c.mu.RLock()
	defer c.mu.RUnlock()

	for topic := range c.handlers {
		client.Subscribe(topic, byte(QoSAtLeastOnce), nil)
	}
}

// Connect establishes connection to the broker.
func (c *Client) Connect(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	token := c.client.Connect()
	return waitToken(token, c.config.ConnectTimeout, "connect to MQTT broker")
}

// Disconnect closes the connection.
func (c *Client) Disconnect() {
	c.client.Disconnect(250)
}

// IsConnected returns whether the client is connected.
func (c *Client) IsConnected() bool {
	return c.client.IsConnected()
}

// Publish sends a message to a topic.
func (c *Client) Publish(ctx context.Context, topic string, payload []byte) error {
	return c.PublishWithOptions(ctx, topic, payload, QoSAtLeastOnce, false)
}

// PublishWithOptions sends a message with specific QoS and retain settings.
func (c *Client) PublishWithOptions(ctx context.Context, topic string, payload []byte, qos QoS, retained bool) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	token := c.client.Publish(topic, byte(qos), retained, payload)
	return waitToken(token, 10*time.Second, "publish message")
}

// Subscribe registers a handler for a topic.
func (c *Client) Subscribe(ctx context.Context, topic string, handler MessageHandler) error {
	return c.SubscribeWithQoS(ctx, topic, QoSAtLeastOnce, handler)
}

// SubscribeWithQoS subscribes with a specific QoS level.
func (c *Client) SubscribeWithQoS(ctx context.Context, topic string, qos QoS, handler MessageHandler) error {
	c.mu.Lock()
	c.handlers[topic] = handler
	c.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return err
	}

	token := c.client.Subscribe(topic, byte(qos), func(client paho.Client, msg paho.Message) {
		handler(&Message{
			Topic:     msg.Topic(),
			Payload:   msg.Payload(),
			QoS:       QoS(msg.Qos()),
			Retained:  msg.Retained(),
			MessageID: msg.MessageID(),
		})
	})

	return waitToken(token, 10*time.Second, "subscribe")
}

// Unsubscribe removes a topic subscription.
func (c *Client) Unsubscribe(ctx context.Context, topic string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c.mu.Lock()
	delete(c.handlers, topic)
	c.mu.Unlock()

	token := c.client.Unsubscribe(topic)
	return waitToken(token, 10*time.Second, "unsubscribe")
}

// SubscribeMultiple subscribes to multiple topics.
func (c *Client) SubscribeMultiple(ctx context.Context, topics map[string]MessageHandler, qos QoS) error {
	c.mu.Lock()
	for topic, handler := range topics {
		c.handlers[topic] = handler
	}
	c.mu.Unlock()

	filters := make(map[string]byte)
	for topic := range topics {
		filters[topic] = byte(qos)
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	token := c.client.SubscribeMultiple(filters, nil)
	return waitToken(token, 10*time.Second, "subscribe to multiple topics")
}

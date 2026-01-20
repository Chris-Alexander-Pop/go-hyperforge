package push

import (
	"context"
	"time"
)

// Sender defines the interface for sending push notifications.
type Sender interface {
	// Send dispatches a single push notification.
	Send(ctx context.Context, msg *Message) error

	// SendBatch dispatches multiple push notifications.
	SendBatch(ctx context.Context, msgs []*Message) error

	// Close releases any resources held by the sender.
	Close() error
}

// Message represents a push notification.
type Message struct {
	// Tokens are the device registration tokens to send to.
	Tokens []string

	// Title is the notification title.
	Title string

	// Body is the notification body text.
	Body string

	// ImageURL is an optional URL to an image to include.
	ImageURL string

	// Data is a map of custom key-value pairs to include in the payload.
	Data map[string]string

	// Platform specifies the target platform (optional, e.g., "ios", "android").
	Platform string

	// Priority specifies the delivery priority (e.g., "high", "normal").
	Priority string

	// TTL specifies the time-to-live for the notification.
	TTL time.Duration

	// Tags are custom tags to associate with the notification.
	Tags map[string]string
}

// Config holds configuration for the Push Sender.
type Config struct {
	// Driver specifies the push backend: "memory", "fcm", "apns".
	Driver string `env:"PUSH_DRIVER" env-default:"memory" validate:"required"`

	// RetryConfig configures the retry behavior for failed sends.
	RetryMax     int           `env:"PUSH_RETRY_MAX" env-default:"3"`
	RetryBackoff time.Duration `env:"PUSH_RETRY_BACKOFF" env-default:"1s"`

	// FCM specific config
	FCMProjectID      string `env:"PUSH_FCM_PROJECT_ID"`
	FCMServiceAccount string `env:"PUSH_FCM_SERVICE_ACCOUNT_JSON"`

	// APNS specific config
	APNSTeamID      string `env:"PUSH_APNS_TEAM_ID"`
	APNSKeyID       string `env:"PUSH_APNS_KEY_ID"`
	APNSKeyFile     string `env:"PUSH_APNS_KEY_FILE"` // Path to .p8 file
	APNSTopic       string `env:"PUSH_APNS_TOPIC"`    // App Bundle ID
	APNSDevelopment bool   `env:"PUSH_APNS_DEVELOPMENT" env-default:"false"`
}

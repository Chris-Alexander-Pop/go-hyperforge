package sms

import (
	"context"
	"time"
)

// Sender defines the interface for sending SMS messages.
type Sender interface {
	// Send dispatches a single SMS message.
	Send(ctx context.Context, msg *Message) error

	// SendBatch dispatches multiple SMS messages.
	SendBatch(ctx context.Context, msgs []*Message) error

	// Close releases any resources held by the sender.
	Close() error
}

// Message represents an SMS message.
type Message struct {
	// From is the sender's phone number or sender ID.
	// If empty, the default sender configured in the adapter should be used.
	From string

	// To is the recipient's phone number.
	// Must be in E.164 format.
	To string

	// Body is the text content of the SMS.
	Body string

	// MediaURL is an optional URL to media to attach (MMS).
	MediaURL string

	// Tags are custom tags to associate with the SMS for tracking/analytics.
	Tags map[string]string
}

// Config holds configuration for the SMS Sender.
type Config struct {
	// Driver specifies the SMS backend: "memory", "twilio", "sns".
	Driver string `env:"SMS_DRIVER" env-default:"memory" validate:"required"`

	// DefaultFrom is the default sender phone number or ID.
	DefaultFrom string `env:"SMS_DEFAULT_FROM"`

	// RetryConfig configures the retry behavior for failed sends.
	RetryMax     int           `env:"SMS_RETRY_MAX" env-default:"3"`
	RetryBackoff time.Duration `env:"SMS_RETRY_BACKOFF" env-default:"1s"`

	// Twilio specific config
	TwilioAccountSID string `env:"SMS_TWILIO_ACCOUNT_SID"`
	TwilioAuthToken  string `env:"SMS_TWILIO_AUTH_TOKEN"`
	TwilioFromNumber string `env:"SMS_TWILIO_FROM_NUMBER"`

	// SNS specific config
	SNSRegion          string `env:"SMS_SNS_REGION"`
	SNSAccessKeyID     string `env:"SMS_SNS_ACCESS_KEY_ID"`
	SNSSecretAccessKey string `env:"SMS_SNS_SECRET_ACCESS_KEY"`
}

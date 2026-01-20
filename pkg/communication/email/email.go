package email

import (
	"context"
	"time"
)

// Sender defines the interface for sending emails.
type Sender interface {
	// Send dispatches a single email message.
	Send(ctx context.Context, msg *Message) error

	// SendBatch dispatches multiple email messages.
	// Implementations may optimize this by sending valid messages and returning
	// an error that aggregates failures, or by failing fast.
	SendBatch(ctx context.Context, msgs []*Message) error

	// Close releases any resources held by the sender.
	Close() error
}

// Message represents an email message.
type Message struct {
	// From is the sender's email address.
	// If empty, the default sender configured in the adapter should be used.
	From string

	// To is the list of recipient email addresses.
	To []string

	// CC is the list of carbon copy recipient email addresses.
	CC []string

	// BCC is the list of blind carbon copy recipient email addresses.
	BCC []string

	// Subject is the email subject line.
	Subject string

	// Body represents the email content.
	// It can be plain text or HTML.
	Body Body

	// Attachments is a list of files to attach to the email.
	Attachments []Attachment

	// ReplyTo is the email address to use for replies.
	ReplyTo string

	// Tags are custom tags to associate with the email for tracking/analytics.
	Tags map[string]string
}

// Body represents the content of an email.
type Body struct {
	// PlainText is the text/plain content.
	PlainText string

	// HTML is the text/html content.
	HTML string
}

// Attachment represents a file attached to an email.
type Attachment struct {
	// Filename is the name of the file as it should appear in the email.
	Filename string

	// Content is the raw bytes of the file.
	Content []byte

	// ContentType is the MIME type of the file (e.g., "application/pdf").
	ContentType string

	// Inline indicates whether the attachment is intended to be displayed inline.
	Inline bool

	// ContentID is the content ID for inline attachments (cid:...).
	ContentID string
}

// Config holds configuration for the Email Sender.
type Config struct {
	// Driver specifies the email backend: "memory", "sendgrid", "ses", "smtp".
	Driver string `env:"EMAIL_DRIVER" env-default:"memory" validate:"required"`

	// DefaultFrom is the default sender email address.
	DefaultFrom string `env:"EMAIL_DEFAULT_FROM" validate:"omitempty,email"`

	// DefaultFromName is the default sender name.
	DefaultFromName string `env:"EMAIL_DEFAULT_FROM_NAME"`

	// RetryConfig configures the retry behavior for failed sends.
	RetryMax     int           `env:"EMAIL_RETRY_MAX" env-default:"3"`
	RetryBackoff time.Duration `env:"EMAIL_RETRY_BACKOFF" env-default:"1s"`

	// SendGrid specific config
	SendGridAPIKey string `env:"EMAIL_SENDGRID_API_KEY"`

	// SES specific config
	SESRegion          string `env:"EMAIL_SES_REGION"`
	SESAccessKeyID     string `env:"EMAIL_SES_ACCESS_KEY_ID"`
	SESSecretAccessKey string `env:"EMAIL_SES_SECRET_ACCESS_KEY"`

	// SMTP specific config
	SMTPHost     string `env:"EMAIL_SMTP_HOST"`
	SMTPPort     int    `env:"EMAIL_SMTP_PORT" env-default:"587"`
	SMTPUsername string `env:"EMAIL_SMTP_USERNAME"`
	SMTPPassword string `env:"EMAIL_SMTP_PASSWORD"`
	SMTPTLS      bool   `env:"EMAIL_SMTP_TLS" env-default:"true"`
}

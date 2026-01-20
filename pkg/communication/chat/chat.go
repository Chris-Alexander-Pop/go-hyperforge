package chat

import (
	"context"
	"time"
)

// Sender defines the interface for sending chat messages.
type Sender interface {
	// Send dispatches a single chat message.
	Send(ctx context.Context, msg *Message) error

	// Close releases any resources held by the sender.
	Close() error
}

// Message represents a chat message.
type Message struct {
	// ChannelID is the ID of the channel to send to.
	ChannelID string

	// UserID is the ID of the user to send to (direct message).
	// One of ChannelID or UserID must be provided.
	UserID string

	// Text is the message content.
	Text string

	// Attachments is a list of attachments (images, files).
	Attachments []Attachment

	// ThreadID is the ID of the thread to reply to (optional).
	ThreadID string

	// Tags are custom tags to associate with the message.
	Tags map[string]string
}

// Attachment represents a file or rich content attached to a chat message.
type Attachment struct {
	// Title is the attachment title.
	Title string

	// Text is the attachment text.
	Text string

	// ImageURL is the URL of an image to display.
	ImageURL string

	// Color is the hex color code for the attachment border (e.g., "#36a64f").
	Color string

	// Fields are key-value pairs to display in a table-like format.
	Fields []AttachmentField
}

// AttachmentField represents a field in an attachment.
type AttachmentField struct {
	Title string
	Value string
	Short bool
}

// Config holds configuration for the Chat Sender.
type Config struct {
	// Driver specifies the chat backend: "memory", "slack", "discord".
	Driver string `env:"CHAT_DRIVER" env-default:"memory" validate:"required"`

	// RetryConfig configures the retry behavior for failed sends.
	RetryMax     int           `env:"CHAT_RETRY_MAX" env-default:"3"`
	RetryBackoff time.Duration `env:"CHAT_RETRY_BACKOFF" env-default:"1s"`

	// Slack specific config
	SlackToken string `env:"CHAT_SLACK_TOKEN"`

	// Discord specific config
	DiscordToken string `env:"CHAT_DISCORD_TOKEN"`
}

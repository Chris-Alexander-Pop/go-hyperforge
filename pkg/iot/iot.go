package iot

import (
	"context"
	"time"
)

// QoS levels for MQTT messages.
type QoS byte

const (
	QoSAtMostOnce  QoS = 0 // Fire and forget
	QoSAtLeastOnce QoS = 1 // Acknowledged delivery
	QoSExactlyOnce QoS = 2 // Ensured delivery
)

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

// Client is the primary MQTT client interface.
// Implementations include adapters/memory and protocols/mqtt (concrete Paho client).
type Client interface {
	// Connect establishes a connection to the broker.
	Connect(ctx context.Context) error

	// Disconnect closes the connection.
	Disconnect()

	// IsConnected reports whether the client is currently connected.
	IsConnected() bool

	// Publish sends a message with QoS at-least-once and retain=false.
	Publish(ctx context.Context, topic string, payload []byte) error

	// PublishWithOptions sends a message with explicit QoS and retain settings.
	PublishWithOptions(ctx context.Context, topic string, payload []byte, qos QoS, retained bool) error

	// Subscribe registers a handler for a topic (QoS at-least-once).
	Subscribe(ctx context.Context, topic string, handler MessageHandler) error

	// SubscribeWithQoS subscribes with a specific QoS level.
	SubscribeWithQoS(ctx context.Context, topic string, qos QoS, handler MessageHandler) error

	// Unsubscribe removes a topic subscription.
	Unsubscribe(ctx context.Context, topic string) error
}

// UpdateState represents the current OTA update state.
type UpdateState string

const (
	StateIdle        UpdateState = "idle"
	StateChecking    UpdateState = "checking"
	StateDownloading UpdateState = "downloading"
	StateVerifying   UpdateState = "verifying"
	StateInstalling  UpdateState = "installing"
	StateRebooting   UpdateState = "rebooting"
	StateFailed      UpdateState = "failed"
	StateComplete    UpdateState = "complete"
)

// UpdateManifest describes an available update.
type UpdateManifest struct {
	Version     string            `json:"version"`
	Description string            `json:"description"`
	ReleaseDate time.Time         `json:"release_date"`
	Files       []UpdateFile      `json:"files"`
	MinVersion  string            `json:"min_version,omitempty"`
	MaxVersion  string            `json:"max_version,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// UpdateFile describes a single update file.
type UpdateFile struct {
	Name     string `json:"name"`
	URL      string `json:"url"`
	Size     int64  `json:"size"`
	SHA256   string `json:"sha256"`
	Required bool   `json:"required"`
}

// UpdateProgress reports download/install progress.
type UpdateProgress struct {
	State           UpdateState `json:"state"`
	CurrentFile     string      `json:"current_file,omitempty"`
	BytesDownloaded int64       `json:"bytes_downloaded"`
	TotalBytes      int64       `json:"total_bytes"`
	Percentage      float64     `json:"percentage"`
	Error           string      `json:"error,omitempty"`
}

// ProgressCallback is called with update progress.
type ProgressCallback func(progress UpdateProgress)

// Updater is the primary OTA update interface.
// Implementations include adapters/memory and device/ota (HTTP downloader).
type Updater interface {
	// CheckForUpdate fetches the manifest and reports whether an update is newer
	// than currentVersion. Uses semantic version comparison.
	CheckForUpdate(ctx context.Context, currentVersion string) (*UpdateManifest, bool, error)

	// DownloadUpdate downloads and checksum-verifies all files in the manifest.
	DownloadUpdate(ctx context.Context, manifest *UpdateManifest) (map[string][]byte, error)

	// ApplyUpdate applies downloaded update files (platform-specific).
	ApplyUpdate(ctx context.Context, files map[string][]byte) error

	// CheckAndApply checks for an update and applies it when available.
	CheckAndApply(ctx context.Context, deviceID, currentVersion string) error

	// GetState returns the current update state.
	GetState() UpdateState

	// SetProgressCallback registers a progress reporter.
	SetProgressCallback(cb ProgressCallback)
}

// Config holds shared IoT package configuration.
type Config struct {
	// Driver selects a backend: "memory", "mqtt", "awsiot".
	Driver string `env:"IOT_DRIVER" env-default:"memory"`
}

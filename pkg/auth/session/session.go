package session

import (
	"context"
	"time"
)

// Config configures the session manager.
type Config struct {
	// Driver specifies the session storage driver: "memory", "redis", etc.
	Driver string `env:"AUTH_SESSION_DRIVER" env-default:"memory"`

	// EncryptionKey is the key used to encrypt session data (optional).
	EncryptionKey string `env:"AUTH_SESSION_ENCRYPTION_KEY"`

	// TTL is the default session duration.
	TTL time.Duration `env:"AUTH_SESSION_TTL" env-default:"24h"`
}

// Session represents a user session.
type Session struct {
	ID        string                 `json:"id"`
	UserID    string                 `json:"user_id"`
	CreatedAt time.Time              `json:"created_at"`
	ExpiresAt time.Time              `json:"expires_at"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// Manager manages user sessions.
type Manager interface {
	// Create creates a new session for a user.
	Create(ctx context.Context, userID string, metadata map[string]interface{}) (*Session, error)

	// Get retrieves a session by ID.
	Get(ctx context.Context, sessionID string) (*Session, error)

	// Delete removes a session.
	Delete(ctx context.Context, sessionID string) error

	// Refresh extends the session expiration.
	Refresh(ctx context.Context, sessionID string) (*Session, error)
}

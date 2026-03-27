package redis

import (
	"context"
	"encoding/json"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/auth/session"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// SessionManager implements session.Manager using Redis.
type SessionManager struct {
	client *redis.Client
	ttl    time.Duration
}

// New creates a new Redis session manager.
func New(client *redis.Client, cfg session.Config) *SessionManager {
	return &SessionManager{
		client: client,
		ttl:    cfg.TTL,
	}
}

func (m *SessionManager) key(sessionID string) string {
	// ⚡ Bolt: Use string concatenation to avoid fmt.Sprintf overhead
	return "auth:session:" + sessionID
}

func (m *SessionManager) Create(ctx context.Context, userID string, metadata map[string]interface{}) (*session.Session, error) {
	id := uuid.NewString()
	now := time.Now()

	s := &session.Session{
		ID:        id,
		UserID:    userID,
		CreatedAt: now,
		ExpiresAt: now.Add(m.ttl),
		Metadata:  metadata,
	}

	data, err := json.Marshal(s)
	if err != nil {
		return nil, errors.Internal("failed to marshal session", err)
	}

	if err := m.client.Set(ctx, m.key(id), data, m.ttl).Err(); err != nil {
		return nil, errors.Internal("failed to save session to redis", err)
	}

	return s, nil
}

func (m *SessionManager) Get(ctx context.Context, sessionID string) (*session.Session, error) {
	data, err := m.client.Get(ctx, m.key(sessionID)).Bytes()
	if err == redis.Nil {
		return nil, errors.NotFound("session not found", nil)
	}
	if err != nil {
		return nil, errors.Internal("failed to get session from redis", err)
	}

	var s session.Session
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, errors.Internal("failed to unmarshal session", err)
	}

	return &s, nil
}

func (m *SessionManager) Delete(ctx context.Context, sessionID string) error {
	if err := m.client.Del(ctx, m.key(sessionID)).Err(); err != nil {
		return errors.Internal("failed to delete session from redis", err)
	}
	return nil
}

func (m *SessionManager) Refresh(ctx context.Context, sessionID string) (*session.Session, error) {
	// Optimistic concurrency could be better, but for sessions usually read-modify-write is okay
	// or just updating TTL. However, we store ExpiresAt inside the struct, so we must update the struct.

	// Transaction?
	key := m.key(sessionID)

	// Watch the key
	var s *session.Session
	err := m.client.Watch(ctx, func(tx *redis.Tx) error {
		data, err := tx.Get(ctx, key).Bytes()
		if err == redis.Nil {
			return errors.NotFound("session not found", nil)
		}
		if err != nil {
			return err
		}

		var current session.Session
		if err := json.Unmarshal(data, &current); err != nil {
			return err
		}

		// Update expiration
		current.ExpiresAt = time.Now().Add(m.ttl)
		s = &current

		newData, err := json.Marshal(s)
		if err != nil {
			return err
		}

		_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
			pipe.Set(ctx, key, newData, m.ttl)
			return nil
		})
		return err
	}, key)

	if err != nil {
		// If watch failed, we could retry, but here we just return error
		if errors.Is(err, redis.TxFailedErr) {
			return nil, errors.Conflict("session update conflict", err)
		}
		// Wrap if it's not already wrapped
		// The error from Watch function might be one we returned (NotFound)
		// We'll trust the error is meaningful or wrap it if generic
		return nil, errors.Internal("failed to refresh session", err)
	}

	return s, nil
}

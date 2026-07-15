package redis

import (
	"context"
	"encoding/json"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/auth"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/auth/session"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/security/crypto"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// SessionManager implements session.Manager using Redis.
type SessionManager struct {
	client    *redis.Client
	ttl       time.Duration
	encryptor *crypto.AESEncryptor
}

// New creates a new Redis session manager.
// When cfg.EncryptionKey is set, the full session payload is encrypted at rest.
func New(client *redis.Client, cfg session.Config) (*SessionManager, error) {
	enc, err := auth.NewAESEncryptorFromKey(cfg.EncryptionKey)
	if err != nil {
		return nil, err
	}
	ttl := cfg.TTL
	if ttl == 0 {
		ttl = 24 * time.Hour
	}
	return &SessionManager{
		client:    client,
		ttl:       ttl,
		encryptor: enc,
	}, nil
}

func (m *SessionManager) key(sessionID string) string {
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

	data, err := m.encode(s)
	if err != nil {
		return nil, err
	}

	if err := m.client.Set(ctx, m.key(id), data, m.ttl).Err(); err != nil {
		return nil, errors.Internal("failed to save session to redis", err)
	}

	return s, nil
}

func (m *SessionManager) Get(ctx context.Context, sessionID string) (*session.Session, error) {
	data, err := m.client.Get(ctx, m.key(sessionID)).Bytes()
	if err == redis.Nil {
		return nil, auth.ErrSessionNotFound
	}
	if err != nil {
		return nil, errors.Internal("failed to get session from redis", err)
	}

	s, err := m.decode(data)
	if err != nil {
		return nil, err
	}
	return s, nil
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
			return auth.ErrSessionNotFound
		}
		if err != nil {
			return err
		}

		current, err := m.decode(data)
		if err != nil {
			return err
		}

		current.ExpiresAt = time.Now().Add(m.ttl)
		s = current

		newData, err := m.encode(s)
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
		if errors.Is(err, redis.TxFailedErr) {
			return nil, errors.Conflict("session update conflict", err)
		}
		if errors.Is(err, auth.ErrSessionNotFound) {
			return nil, err
		}
		return nil, errors.Internal("failed to refresh session", err)
	}

	return s, nil
}

func (m *SessionManager) encode(s *session.Session) ([]byte, error) {
	raw, err := json.Marshal(s)
	if err != nil {
		return nil, errors.Internal("failed to marshal session", err)
	}
	if m.encryptor == nil {
		return raw, nil
	}
	enc, err := m.encryptor.EncryptString(string(raw))
	if err != nil {
		return nil, errors.Internal("failed to encrypt session", err)
	}
	wrapped, err := json.Marshal(map[string]string{"_enc": enc})
	if err != nil {
		return nil, errors.Internal("failed to marshal encrypted session", err)
	}
	return wrapped, nil
}

func (m *SessionManager) decode(data []byte) (*session.Session, error) {
	if m.encryptor != nil {
		var wrap map[string]string
		if err := json.Unmarshal(data, &wrap); err == nil {
			if enc, ok := wrap["_enc"]; ok {
				plain, err := m.encryptor.DecryptString(enc)
				if err != nil {
					return nil, errors.Internal("failed to decrypt session", err)
				}
				data = []byte(plain)
			}
		}
	}
	var s session.Session
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, errors.Internal("failed to unmarshal session", err)
	}
	return &s, nil
}

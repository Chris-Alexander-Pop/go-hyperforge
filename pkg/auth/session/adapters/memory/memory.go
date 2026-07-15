package memory

import (
	"context"
	"encoding/json"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/auth"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/auth/session"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/concurrency"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/security/crypto"
	"github.com/google/uuid"
)

// SessionManager implements session.Manager using in-memory storage.
type SessionManager struct {
	sessions  map[string]*session.Session
	mu        *concurrency.SmartRWMutex
	ttl       time.Duration
	encryptor *crypto.AESEncryptor
}

// New creates a new in-memory session manager.
// When cfg.EncryptionKey is set, session metadata is encrypted at rest in the store.
func New(cfg session.Config) (*SessionManager, error) {
	enc, err := auth.NewAESEncryptorFromKey(cfg.EncryptionKey)
	if err != nil {
		return nil, err
	}
	ttl := cfg.TTL
	if ttl == 0 {
		ttl = 24 * time.Hour
	}
	return &SessionManager{
		sessions: make(map[string]*session.Session),
		mu: concurrency.NewSmartRWMutex(concurrency.MutexConfig{
			Name: "memory-session-manager",
		}),
		ttl:       ttl,
		encryptor: enc,
	}, nil
}

func (m *SessionManager) Create(ctx context.Context, userID string, metadata map[string]interface{}) (*session.Session, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	id := uuid.NewString()
	now := time.Now()
	meta, err := m.sealMetadata(metadata)
	if err != nil {
		return nil, err
	}
	s := &session.Session{
		ID:        id,
		UserID:    userID,
		CreatedAt: now,
		ExpiresAt: now.Add(m.ttl),
		Metadata:  meta,
	}

	m.sessions[id] = cloneSession(s)
	out := cloneSession(s)
	out.Metadata, err = m.openMetadata(out.Metadata)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (m *SessionManager) Get(ctx context.Context, sessionID string) (*session.Session, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()

	s, ok := m.sessions[sessionID]
	if !ok {
		return nil, auth.ErrSessionNotFound
	}

	if time.Now().After(s.ExpiresAt) {
		return nil, auth.ErrSessionNotFound
	}

	out := cloneSession(s)
	meta, err := m.openMetadata(out.Metadata)
	if err != nil {
		return nil, err
	}
	out.Metadata = meta
	return out, nil
}

func (m *SessionManager) Delete(ctx context.Context, sessionID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, sessionID)
	return nil
}

func (m *SessionManager) Refresh(ctx context.Context, sessionID string) (*session.Session, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	s, ok := m.sessions[sessionID]
	if !ok {
		return nil, auth.ErrSessionNotFound
	}

	if time.Now().After(s.ExpiresAt) {
		delete(m.sessions, sessionID)
		return nil, auth.ErrSessionNotFound
	}

	s.ExpiresAt = time.Now().Add(m.ttl)
	out := cloneSession(s)
	meta, err := m.openMetadata(out.Metadata)
	if err != nil {
		return nil, err
	}
	out.Metadata = meta
	return out, nil
}

func (m *SessionManager) sealMetadata(metadata map[string]interface{}) (map[string]interface{}, error) {
	if m.encryptor == nil || metadata == nil {
		return metadata, nil
	}
	raw, err := json.Marshal(metadata)
	if err != nil {
		return nil, errors.Internal("failed to marshal session metadata", err)
	}
	enc, err := m.encryptor.EncryptString(string(raw))
	if err != nil {
		return nil, errors.Internal("failed to encrypt session metadata", err)
	}
	return map[string]interface{}{"_enc": enc}, nil
}

func (m *SessionManager) openMetadata(metadata map[string]interface{}) (map[string]interface{}, error) {
	if m.encryptor == nil || metadata == nil {
		return metadata, nil
	}
	enc, ok := metadata["_enc"].(string)
	if !ok {
		return metadata, nil
	}
	plain, err := m.encryptor.DecryptString(enc)
	if err != nil {
		return nil, errors.Internal("failed to decrypt session metadata", err)
	}
	var out map[string]interface{}
	if err := json.Unmarshal([]byte(plain), &out); err != nil {
		return nil, errors.Internal("failed to unmarshal session metadata", err)
	}
	return out, nil
}

func cloneSession(s *session.Session) *session.Session {
	if s == nil {
		return nil
	}
	out := *s
	if s.Metadata != nil {
		out.Metadata = make(map[string]interface{}, len(s.Metadata))
		for k, v := range s.Metadata {
			out.Metadata[k] = v
		}
	}
	return &out
}

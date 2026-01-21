package memory

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/auth/session"
	"github.com/chris-alexander-pop/system-design-library/pkg/concurrency"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"github.com/google/uuid"
)

// SessionManager implements session.Manager using in-memory storage.
type SessionManager struct {
	sessions map[string]*session.Session
	mu       *concurrency.SmartRWMutex
	ttl      time.Duration
}

// New creates a new in-memory session manager.
func New(cfg session.Config) *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*session.Session),
		mu: concurrency.NewSmartRWMutex(concurrency.MutexConfig{
			Name: "memory-session-manager",
		}),
		ttl: cfg.TTL,
	}
}

func (m *SessionManager) Create(ctx context.Context, userID string, metadata map[string]interface{}) (*session.Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	id := uuid.NewString()
	now := time.Now()
	s := &session.Session{
		ID:        id,
		UserID:    userID,
		CreatedAt: now,
		ExpiresAt: now.Add(m.ttl),
		Metadata:  metadata,
	}

	m.sessions[id] = s
	return s, nil
}

func (m *SessionManager) Get(ctx context.Context, sessionID string) (*session.Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	s, ok := m.sessions[sessionID]
	if !ok {
		return nil, errors.NotFound("session not found", nil)
	}

	if time.Now().After(s.ExpiresAt) {
		delete(m.sessions, sessionID)
		return nil, errors.NotFound("session expired", nil)
	}

	return s, nil
}

func (m *SessionManager) Delete(ctx context.Context, sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.sessions[sessionID]; !ok {
		// Idempotent delete
		return nil
	}
	delete(m.sessions, sessionID)
	return nil
}

func (m *SessionManager) Refresh(ctx context.Context, sessionID string) (*session.Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	s, ok := m.sessions[sessionID]
	if !ok {
		return nil, errors.NotFound("session not found", nil)
	}

	if time.Now().After(s.ExpiresAt) {
		delete(m.sessions, sessionID)
		return nil, errors.NotFound("session expired", nil)
	}

	s.ExpiresAt = time.Now().Add(m.ttl)
	return s, nil
}

package memory

import (
	"context"

	"github.com/chris-alexander-pop/system-design-library/pkg/auth/webauthn"
	"github.com/chris-alexander-pop/system-design-library/pkg/concurrency"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
)

// WebAuthnService implements webauthn.Service using in-memory storage (mock).
// Note: Real WebAuthn logic requires a compliant library like github.com/go-webauthn/webauthn.
// This is a stub implementation to satisfy the interface.
type WebAuthnService struct {
	credentials map[string][]webauthn.Credential // userID -> credentials
	mu          *concurrency.SmartRWMutex
}

// New creates a new in-memory WebAuthn service.
func New(cfg webauthn.Config) *WebAuthnService {
	return &WebAuthnService{
		credentials: make(map[string][]webauthn.Credential),
		mu:          concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "memory-webauthn"}),
	}
}

func (s *WebAuthnService) BeginRegistration(ctx context.Context, user webauthn.User) (interface{}, error) {
	// Stub
	return map[string]string{
		"challenge": "mock-challenge",
		"rp":        "mock-rp",
	}, nil
}

func (s *WebAuthnService) FinishRegistration(ctx context.Context, user webauthn.User, sessionData interface{}, responseData interface{}) (*webauthn.Credential, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Mock credential creation
	cred := webauthn.Credential{
		ID:              []byte("mock-cred-id"),
		PublicKey:       []byte("mock-pub-key"),
		AttestationType: "none",
	}

	userID := user.WebAuthnName() // Simplified mapping
	s.credentials[userID] = append(s.credentials[userID], cred)

	return &cred, nil
}

func (s *WebAuthnService) BeginLogin(ctx context.Context, user webauthn.User) (interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	userID := user.WebAuthnName() // Simplified
	if _, ok := s.credentials[userID]; !ok {
		return nil, errors.NotFound("user not found or no credentials", nil)
	}

	return map[string]string{
		"challenge": "mock-login-challenge",
	}, nil
}

func (s *WebAuthnService) FinishLogin(ctx context.Context, user webauthn.User, sessionData interface{}, responseData interface{}) (*webauthn.Credential, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	userID := user.WebAuthnName()
	creds, ok := s.credentials[userID]
	if !ok || len(creds) == 0 {
		return nil, errors.NotFound("no credentials found", nil)
	}

	// Just return the first one as "success"
	return &creds[0], nil
}

package memory

import (
	"context"
	"crypto/rand"
	"encoding/base64"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/concurrency"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/security/secrets"
)

// SecretManager implements secrets.SecretManager using in-memory storage.
type SecretManager struct {
	secrets map[string]string
	mu      *concurrency.SmartRWMutex
}

// Ensure SecretManager implements secrets.SecretManager.
var _ secrets.SecretManager = (*SecretManager)(nil)

// New creates a new in-memory secret manager.
func New() secrets.SecretManager {
	return &SecretManager{
		secrets: make(map[string]string),
		mu:      concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "memory-secret-manager"}),
	}
}

func (m *SecretManager) Get(ctx context.Context, name string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if name == "" {
		return "", secrets.ErrInvalidArgument
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	val, ok := m.secrets[name]
	if !ok {
		return "", secrets.ErrNotFound
	}
	return val, nil
}

func (m *SecretManager) Set(ctx context.Context, name, value string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if name == "" {
		return secrets.ErrInvalidArgument
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.secrets[name] = value
	return nil
}

func (m *SecretManager) Rotate(ctx context.Context, name, newValue string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if name == "" {
		return "", secrets.ErrInvalidArgument
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.secrets[name]; !ok {
		return "", secrets.ErrNotFound
	}

	if newValue == "" {
		buf := make([]byte, 32)
		if _, err := rand.Read(buf); err != nil {
			return "", secrets.ErrRotateFailed
		}
		newValue = base64.RawURLEncoding.EncodeToString(buf)
	}

	m.secrets[name] = newValue
	return newValue, nil
}

// Delete removes a secret. This is an extension beyond secrets.SecretManager
// (mirrors cloud adapters such as Azure Key Vault).
func (m *SecretManager) Delete(ctx context.Context, name string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if name == "" {
		return secrets.ErrInvalidArgument
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.secrets[name]; !ok {
		return secrets.ErrNotFound
	}
	delete(m.secrets, name)
	return nil
}

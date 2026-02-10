package memory

import (
	"context"

	"github.com/chris-alexander-pop/system-design-library/pkg/concurrency"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"github.com/chris-alexander-pop/system-design-library/pkg/security/secrets"
)

// SecretManager implements secrets.SecretManager using in-memory storage.
type SecretManager struct {
	secrets map[string]string
	mu      *concurrency.SmartRWMutex
}

// New creates a new in-memory secret manager.
func New() secrets.SecretManager {
	return &SecretManager{
		secrets: make(map[string]string),
		mu:      concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "memory-secret-manager"}),
	}
}

func (m *SecretManager) Get(ctx context.Context, name string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	val, ok := m.secrets[name]
	if !ok {
		return "", errors.NotFound("secret not found", nil)
	}
	return val, nil
}

func (m *SecretManager) Set(ctx context.Context, name, value string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.secrets[name] = value
	return nil
}

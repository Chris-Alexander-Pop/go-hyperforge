package memory

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/concurrency"
	"github.com/chris-alexander-pop/system-design-library/pkg/security/waf"
	"github.com/google/uuid"
)

// Manager implements waf.Manager using in-memory storage.
type Manager struct {
	blockedIPs map[string]waf.Rule
	mu         *concurrency.SmartRWMutex
}

// New creates a new in-memory WAF manager.
func New() *Manager {
	return &Manager{
		blockedIPs: make(map[string]waf.Rule),
		mu:         concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "memory-waf-manager"}),
	}
}

func (m *Manager) BlockIP(ctx context.Context, ip, reason string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.blockedIPs[ip] = waf.Rule{
		ID:        uuid.NewString(),
		IP:        ip,
		Action:    "block",
		Reason:    reason,
		ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
	}
	return nil
}

func (m *Manager) AllowIP(ctx context.Context, ip string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.blockedIPs, ip)
	return nil
}

func (m *Manager) GetRules(ctx context.Context) ([]waf.Rule, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	rules := make([]waf.Rule, 0, len(m.blockedIPs))
	for _, rule := range m.blockedIPs {
		rules = append(rules, rule)
	}
	return rules, nil
}

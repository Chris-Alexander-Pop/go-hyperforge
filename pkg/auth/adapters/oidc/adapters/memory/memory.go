package memory

import (
	"context"
	"strings"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/auth"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/auth/adapters/oidc"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/concurrency"
	"golang.org/x/oauth2"
)

// Exchanger is an in-memory CodeExchanger for tests.
type Exchanger struct {
	mu       *concurrency.SmartRWMutex
	codes    map[string]*oidc.TokenSet
	authURL  string
	clientID string
}

// New creates a memory CodeExchanger.
func New(clientID string) *Exchanger {
	return &Exchanger{
		mu:       concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "oidc-memory-exchanger"}),
		codes:    make(map[string]*oidc.TokenSet),
		authURL:  "https://example.test/authorize",
		clientID: clientID,
	}
}

// Seed registers an authorization code that Exchange will redeem.
func (e *Exchanger) Seed(code string, ts *oidc.TokenSet) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.codes[code] = ts
}

// AuthCodeURL returns a deterministic test authorization URL.
func (e *Exchanger) AuthCodeURL(state string, opts ...oauth2.AuthCodeOption) (string, error) {
	_ = opts
	return e.authURL + "?client_id=" + e.clientID + "&state=" + state, nil
}

// Exchange redeems a previously seeded code.
func (e *Exchanger) Exchange(ctx context.Context, code string, opts ...oauth2.AuthCodeOption) (*oidc.TokenSet, error) {
	_ = opts
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(code) == "" {
		return nil, auth.ErrExchangeFailed
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	ts, ok := e.codes[code]
	if !ok {
		return nil, auth.ErrExchangeFailed
	}
	delete(e.codes, code)
	if ts.Claims == nil {
		ts.Claims = &auth.Claims{
			Subject:   "memory-user",
			Issuer:    "memory-oidc",
			Audience:  []string{e.clientID},
			ExpiresAt: time.Now().Add(time.Hour).Unix(),
			IssuedAt:  time.Now().Unix(),
		}
	}
	return ts, nil
}

var _ oidc.CodeExchanger = (*Exchanger)(nil)

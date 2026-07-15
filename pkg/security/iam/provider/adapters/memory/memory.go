package memory

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/concurrency"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/security/iam"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/security/iam/provider"
	"github.com/google/uuid"
)

type userEntry struct {
	user     iam.User
	password string
}

// MemoryIdentityProvider is an in-memory implementation of IdentityProvider.
type MemoryIdentityProvider struct {
	users  map[string]userEntry // map[username]userEntry
	tokens map[string]string    // map[token]username (simple validation)
	mu     *concurrency.SmartRWMutex
}

// New creates a new MemoryIdentityProvider.
func New() *MemoryIdentityProvider {
	return &MemoryIdentityProvider{
		users:  make(map[string]userEntry),
		tokens: make(map[string]string),
		mu: concurrency.NewSmartRWMutex(concurrency.MutexConfig{
			Name: "memory-idp",
		}),
	}
}

func (p *MemoryIdentityProvider) Authenticate(ctx context.Context, creds iam.Credentials) (*iam.User, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	entry, ok := p.users[creds.Username]
	if !ok {
		return nil, provider.ErrInvalidCredentials
	}

	// In real world, bcrypt compare needed here. Memory uses plaintext for simplicity.
	if entry.password != creds.Password {
		return nil, provider.ErrInvalidCredentials
	}

	u := entry.user
	return &u, nil
}

func (p *MemoryIdentityProvider) IssueToken(ctx context.Context, user *iam.User, scopes []string) (*iam.Token, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Simple UUID token
	accessToken := uuid.NewString()

	p.tokens[accessToken] = user.Username

	token := &iam.Token{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   3600,
		IssuedAt:    time.Now(),
	}
	return token, nil
}

func (p *MemoryIdentityProvider) ValidateToken(ctx context.Context, token string) (*iam.User, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	username, ok := p.tokens[token]
	if !ok {
		return nil, provider.ErrTokenExpired
	}

	entry, ok := p.users[username]
	if !ok {
		// Should not happen if data consistent
		return nil, provider.ErrUserNotFound
	}

	u := entry.user
	return &u, nil
}

func (p *MemoryIdentityProvider) RevokeToken(ctx context.Context, token string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	delete(p.tokens, token)
	return nil
}

func (p *MemoryIdentityProvider) CreateUser(ctx context.Context, user iam.User, password string) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, ok := p.users[user.Username]; ok {
		return "", provider.ErrUserAlreadyExists
	}

	id := uuid.NewString()
	user.ID = id
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	p.users[user.Username] = userEntry{
		user:     user,
		password: password,
	}

	return id, nil
}

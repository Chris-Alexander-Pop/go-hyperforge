package store

import (
	"context"
	"sync"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/security/crypto"
	"github.com/google/uuid"

	authpassword "github.com/chris-alexander-pop/go-hyperforge/pkg/auth/password"
)

// Account is a registered credential record.
type Account struct {
	UserID string
	Email  string
}

// Store holds email→account mappings and password hashes.
type Store struct {
	mu        sync.RWMutex
	byEmail   map[string]Account
	passwords *authpassword.Store
}

// New creates an in-memory credential store.
func New() *Store {
	return &Store{
		byEmail:   make(map[string]Account),
		passwords: authpassword.New(crypto.DefaultHashConfig()),
	}
}

// Register creates a new account. Returns Conflict if email exists.
func (s *Store) Register(ctx context.Context, email, password string) (Account, error) {
	if email == "" || password == "" {
		return Account{}, errors.InvalidArgument("email and password are required", nil)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.byEmail[email]; exists {
		return Account{}, errors.Conflict("email already registered", nil)
	}

	if err := s.passwords.Set(ctx, email, password); err != nil {
		return Account{}, err
	}

	acct := Account{UserID: uuid.NewString(), Email: email}
	s.byEmail[email] = acct
	return acct, nil
}

// Authenticate verifies credentials and returns the account.
func (s *Store) Authenticate(ctx context.Context, email, password string) (Account, error) {
	if _, err := s.passwords.Authenticate(ctx, email, password); err != nil {
		return Account{}, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	acct, ok := s.byEmail[email]
	if !ok {
		return Account{}, errors.Unauthorized("invalid credentials", nil)
	}
	return acct, nil
}

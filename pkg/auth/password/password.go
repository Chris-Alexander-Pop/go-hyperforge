// Package password provides local username/password authentication using
// pkg/security/crypto.Hasher (Argon2id by default).
//
// Use Store as an oauth2.PasswordAuthenticator or standalone credential vault
// for memory/test identity providers. It never stores plaintext passwords.
package password

import (
	"context"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/security/crypto"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/concurrency"
)

// Store holds username → password-hash mappings.
type Store struct {
	mu     *concurrency.SmartRWMutex
	hasher *crypto.Hasher
	hashes map[string]string
}

// New creates a password store with the given hasher config.
// Pass crypto.DefaultHashConfig() for Argon2id defaults.
func New(cfg crypto.HashConfig) *Store {
	return &Store{
		hasher: crypto.NewHasher(cfg),
		hashes: make(map[string]string),
		mu:     concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "auth-password"}),
	}
}

// Set hashes and stores a password for username (overwrites existing).
func (s *Store) Set(ctx context.Context, username, password string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if username == "" || password == "" {
		return errors.InvalidArgument("username and password are required", nil)
	}
	hash, err := s.hasher.Hash(password)
	if err != nil {
		return errors.Internal("failed to hash password", err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.hashes[username] = hash
	return nil
}

// Authenticate verifies username/password and returns the username as subject.
// Implements oauth2.PasswordAuthenticator.
func (s *Store) Authenticate(ctx context.Context, username, password string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	s.mu.RLock()
	hash, ok := s.hashes[username]
	s.mu.RUnlock()
	if !ok {
		return "", errors.Unauthorized("invalid credentials", nil)
	}
	ok, err := s.hasher.Verify(password, hash)
	if err != nil {
		return "", errors.Internal("failed to verify password", err)
	}
	if !ok {
		return "", errors.Unauthorized("invalid credentials", nil)
	}
	return username, nil
}

// Hashed reports whether a username has a stored password hash (test helper).
func (s *Store) Hashed(username string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	h, ok := s.hashes[username]
	return ok && h != "" && h != username
}

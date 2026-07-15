package memory

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/concurrency"
	"github.com/chris-alexander-pop/system-design-library/pkg/web3"
)

// Ensure compile-time interface compliance.
var _ web3.Verifier = (*Verifier)(nil)

// Verifier is an in-memory SIWE verifier for tests.
// It does not perform cryptographic signature recovery. A signature is accepted
// when it equals MemorySignature(message) (see MemorySignature).
type Verifier struct {
	mu         *concurrency.SmartRWMutex
	usedNonces map[string]time.Time
}

// NewVerifier creates a race-safe in-memory SIWE verifier.
func NewVerifier() *Verifier {
	return &Verifier{
		mu:         concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "web3-memory-siwe"}),
		usedNonces: make(map[string]time.Time),
	}
}

// MemorySignature returns the deterministic "signature" accepted by Verifier.
func MemorySignature(message *web3.SIWEMessage) string {
	if message == nil {
		return ""
	}
	return "memory:" + message.Nonce + ":" + strings.ToLower(message.Address)
}

// GenerateNonce creates a random nonce for SIWE.
func (v *Verifier) GenerateNonce() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", web3.ErrInvalidConfig("failed to generate nonce", err)
	}
	return hex.EncodeToString(bytes), nil
}

// CreateMessage creates a new SIWE message with a fresh nonce.
func (v *Verifier) CreateMessage(domain, address, uri, statement string, chainID int) (*web3.SIWEMessage, error) {
	nonce, err := v.GenerateNonce()
	if err != nil {
		return nil, err
	}
	return &web3.SIWEMessage{
		Domain:    domain,
		Address:   address,
		Statement: statement,
		URI:       uri,
		Version:   "1",
		ChainID:   chainID,
		Nonce:     nonce,
		IssuedAt:  time.Now().UTC(),
	}, nil
}

// Verify checks time bounds, nonce reuse, and the memory signature convention.
func (v *Verifier) Verify(ctx context.Context, message *web3.SIWEMessage, signature string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if message == nil {
		return false, web3.ErrInvalidSignature("message is required", nil)
	}
	if message.ExpirationTime != nil && time.Now().After(*message.ExpirationTime) {
		return false, web3.ErrMessageExpired()
	}
	if message.NotBefore != nil && time.Now().Before(*message.NotBefore) {
		return false, web3.ErrMessageNotYetValid()
	}

	v.mu.Lock()
	defer v.mu.Unlock()

	if _, used := v.usedNonces[message.Nonce]; used {
		return false, web3.ErrNonceReused(message.Nonce)
	}

	expected := MemorySignature(message)
	if signature != expected {
		if signature == "" {
			return false, web3.ErrInvalidSignature("invalid signature format", fmt.Errorf("empty signature"))
		}
		return false, nil
	}

	v.usedNonces[message.Nonce] = time.Now()
	return true, nil
}

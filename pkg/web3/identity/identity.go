// Package identity provides Web3 identity and wallet authentication.
//
// Supports WalletConnect, Sign-In with Ethereum (SIWE), and DIDs.
//
// Usage:
//
//	import "github.com/chris-alexander-pop/system-design-library/pkg/web3/identity"
//
//	verifier := identity.NewSIWEVerifier()
//	valid, err := verifier.Verify(ctx, message, signature)
package identity

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"time"

	pkgerrors "github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)

var didRe = regexp.MustCompile(`^did:([a-z0-9]+):([a-zA-Z0-9._-]+)(?:/([^?#]*))?(?:\?([^#]*))?(?:#(.*))?$`)

// SIWEMessage represents a Sign-In with Ethereum message.
type SIWEMessage struct {
	Domain         string
	Address        string
	Statement      string
	URI            string
	Version        string
	ChainID        int
	Nonce          string
	IssuedAt       time.Time
	ExpirationTime *time.Time
	NotBefore      *time.Time
	RequestID      string
	Resources      []string
}

// String formats the SIWE message for signing.
func (m *SIWEMessage) String() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("%s wants you to sign in with your Ethereum account:\n", m.Domain))
	sb.WriteString(m.Address + "\n\n")

	if m.Statement != "" {
		sb.WriteString(m.Statement + "\n\n")
	}

	sb.WriteString(fmt.Sprintf("URI: %s\n", m.URI))
	sb.WriteString(fmt.Sprintf("Version: %s\n", m.Version))
	sb.WriteString(fmt.Sprintf("Chain ID: %d\n", m.ChainID))
	sb.WriteString(fmt.Sprintf("Nonce: %s\n", m.Nonce))
	sb.WriteString(fmt.Sprintf("Issued At: %s", m.IssuedAt.UTC().Format(time.RFC3339)))

	if m.ExpirationTime != nil {
		sb.WriteString(fmt.Sprintf("\nExpiration Time: %s", m.ExpirationTime.UTC().Format(time.RFC3339)))
	}
	if m.NotBefore != nil {
		sb.WriteString(fmt.Sprintf("\nNot Before: %s", m.NotBefore.UTC().Format(time.RFC3339)))
	}
	if m.RequestID != "" {
		sb.WriteString(fmt.Sprintf("\nRequest ID: %s", m.RequestID))
	}
	if len(m.Resources) > 0 {
		sb.WriteString("\nResources:")
		for _, r := range m.Resources {
			sb.WriteString(fmt.Sprintf("\n- %s", r))
		}
	}

	return sb.String()
}

// SIWEVerifier verifies Sign-In with Ethereum signatures.
type SIWEVerifier struct {
	usedNonces map[string]time.Time
}

// NewSIWEVerifier creates a new SIWE verifier.
func NewSIWEVerifier() *SIWEVerifier {
	return &SIWEVerifier{
		usedNonces: make(map[string]time.Time),
	}
}

// GenerateNonce creates a random nonce for SIWE.
func (v *SIWEVerifier) GenerateNonce() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", pkgerrors.Internal("failed to generate nonce", err)
	}
	return hex.EncodeToString(bytes), nil
}

// CreateMessage creates a new SIWE message.
func (v *SIWEVerifier) CreateMessage(domain, address, uri, statement string, chainID int) (*SIWEMessage, error) {
	nonce, err := v.GenerateNonce()
	if err != nil {
		return nil, err
	}

	return &SIWEMessage{
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

// Verify verifies a SIWE signature.
func (v *SIWEVerifier) Verify(ctx context.Context, message *SIWEMessage, signature string) (bool, error) {
	// Check expiration
	if message.ExpirationTime != nil && time.Now().After(*message.ExpirationTime) {
		return false, pkgerrors.InvalidArgument("message expired", nil)
	}

	// Check not before
	if message.NotBefore != nil && time.Now().Before(*message.NotBefore) {
		return false, pkgerrors.InvalidArgument("message not yet valid", nil)
	}

	// Check nonce hasn't been used
	if _, used := v.usedNonces[message.Nonce]; used {
		return false, pkgerrors.InvalidArgument("nonce already used", nil)
	}

	// Verify signature
	msgStr := message.String()
	prefixedMsg := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(msgStr), msgStr)
	msgHash := crypto.Keccak256Hash([]byte(prefixedMsg))

	sigBytes, err := hexutil.Decode(signature)
	if err != nil {
		return false, pkgerrors.InvalidArgument("invalid signature format", err)
	}

	// Adjust v value for recovery
	if len(sigBytes) == 65 {
		if sigBytes[64] >= 27 {
			sigBytes[64] -= 27
		}
	}

	pubKey, err := crypto.SigToPub(msgHash.Bytes(), sigBytes)
	if err != nil {
		return false, pkgerrors.InvalidArgument("failed to recover public key", err)
	}

	recoveredAddr := crypto.PubkeyToAddress(*pubKey)
	expectedAddr := common.HexToAddress(message.Address)

	if recoveredAddr != expectedAddr {
		return false, nil
	}

	// Mark nonce as used
	v.usedNonces[message.Nonce] = time.Now()

	return true, nil
}

// VerifySignature verifies a simple Ethereum signature.
func VerifySignature(message, signature, address string) (bool, error) {
	prefixedMsg := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(message), message)
	msgHash := crypto.Keccak256Hash([]byte(prefixedMsg))

	sigBytes, err := hexutil.Decode(signature)
	if err != nil {
		return false, pkgerrors.InvalidArgument("invalid signature", err)
	}

	if len(sigBytes) == 65 && sigBytes[64] >= 27 {
		sigBytes[64] -= 27
	}

	pubKey, err := crypto.SigToPub(msgHash.Bytes(), sigBytes)
	if err != nil {
		return false, pkgerrors.InvalidArgument("failed to recover public key", err)
	}

	recoveredAddr := crypto.PubkeyToAddress(*pubKey)
	expectedAddr := common.HexToAddress(address)

	return recoveredAddr == expectedAddr, nil
}

// DID represents a Decentralized Identifier.
type DID struct {
	Method     string
	Identifier string
	Path       string
	Query      string
	Fragment   string
}

// Parse parses a DID string.
func ParseDID(did string) (*DID, error) {
	// Basic DID regex: did:method:identifier
	matches := didRe.FindStringSubmatch(did)
	if matches == nil {
		return nil, pkgerrors.InvalidArgument("invalid DID format", nil)
	}

	d := &DID{
		Method:     matches[1],
		Identifier: matches[2],
	}
	if len(matches) > 3 {
		d.Path = matches[3]
	}
	if len(matches) > 4 {
		d.Query = matches[4]
	}
	if len(matches) > 5 {
		d.Fragment = matches[5]
	}

	return d, nil
}

// String returns the DID as a string.
func (d *DID) String() string {
	result := fmt.Sprintf("did:%s:%s", d.Method, d.Identifier)
	if d.Path != "" {
		result += "/" + d.Path
	}
	if d.Query != "" {
		result += "?" + d.Query
	}
	if d.Fragment != "" {
		result += "#" + d.Fragment
	}
	return result
}

// EthereumDID creates a DID from an Ethereum address.
func EthereumDID(address string) *DID {
	return &DID{
		Method:     "ethr",
		Identifier: strings.ToLower(address),
	}
}

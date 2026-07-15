package identity

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/concurrency"
	"github.com/chris-alexander-pop/system-design-library/pkg/web3"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)

// Ensure SIWEVerifier implements web3.Verifier.
var _ web3.Verifier = (*SIWEVerifier)(nil)

// SIWEMessage is an alias for web3.SIWEMessage.
type SIWEMessage = web3.SIWEMessage

// FormatSIWE formats a SIWE message for EIP-191 personal_sign.
func FormatSIWE(m *web3.SIWEMessage) string {
	if m == nil {
		return ""
	}
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
// Nonce consumption is race-safe via pkg/concurrency.SmartRWMutex.
type SIWEVerifier struct {
	mu         *concurrency.SmartRWMutex
	usedNonces map[string]time.Time
}

// NewSIWEVerifier creates a new SIWE verifier with a race-safe nonce store.
func NewSIWEVerifier() *SIWEVerifier {
	return &SIWEVerifier{
		mu:         concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "web3-siwe-nonces"}),
		usedNonces: make(map[string]time.Time),
	}
}

// GenerateNonce creates a random nonce for SIWE.
func (v *SIWEVerifier) GenerateNonce() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", web3.ErrInvalidConfig("failed to generate nonce", err)
	}
	return hex.EncodeToString(bytes), nil
}

// CreateMessage creates a new SIWE message.
func (v *SIWEVerifier) CreateMessage(domain, address, uri, statement string, chainID int) (*web3.SIWEMessage, error) {
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

// Verify verifies a SIWE signature (EIP-191 personal_sign recovery).
func (v *SIWEVerifier) Verify(ctx context.Context, message *web3.SIWEMessage, signature string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if message == nil {
		return false, web3.ErrInvalidSignature("message is required", nil)
	}

	// Check expiration
	if message.ExpirationTime != nil && time.Now().After(*message.ExpirationTime) {
		return false, web3.ErrMessageExpired()
	}

	// Check not before
	if message.NotBefore != nil && time.Now().Before(*message.NotBefore) {
		return false, web3.ErrMessageNotYetValid()
	}

	v.mu.Lock()
	defer v.mu.Unlock()

	// Check nonce hasn't been used
	if _, used := v.usedNonces[message.Nonce]; used {
		return false, web3.ErrNonceReused(message.Nonce)
	}

	// Verify signature
	msgStr := FormatSIWE(message)
	prefixedMsg := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(msgStr), msgStr)
	msgHash := crypto.Keccak256Hash([]byte(prefixedMsg))

	sigBytes, err := hexutil.Decode(signature)
	if err != nil {
		return false, web3.ErrInvalidSignature("invalid signature format", err)
	}

	// Adjust v value for recovery
	if len(sigBytes) == 65 {
		if sigBytes[64] >= 27 {
			sigBytes[64] -= 27
		}
	}

	pubKey, err := crypto.SigToPub(msgHash.Bytes(), sigBytes)
	if err != nil {
		return false, web3.ErrInvalidSignature("failed to recover public key", err)
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

// VerifySignature verifies a simple Ethereum personal_sign signature.
func VerifySignature(message, signature, address string) (bool, error) {
	prefixedMsg := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(message), message)
	msgHash := crypto.Keccak256Hash([]byte(prefixedMsg))

	sigBytes, err := hexutil.Decode(signature)
	if err != nil {
		return false, web3.ErrInvalidSignature("invalid signature", err)
	}

	if len(sigBytes) == 65 && sigBytes[64] >= 27 {
		sigBytes[64] -= 27
	}

	pubKey, err := crypto.SigToPub(msgHash.Bytes(), sigBytes)
	if err != nil {
		return false, web3.ErrInvalidSignature("failed to recover public key", err)
	}

	recoveredAddr := crypto.PubkeyToAddress(*pubKey)
	expectedAddr := common.HexToAddress(address)

	return recoveredAddr == expectedAddr, nil
}

// SignPersonal signs a message with EIP-191 personal_sign and returns a hex signature.
// Intended for tests and local tooling; production signing belongs in wallets.
func SignPersonal(message string, privateKeyHex string) (string, error) {
	key, err := crypto.HexToECDSA(strings.TrimPrefix(privateKeyHex, "0x"))
	if err != nil {
		return "", web3.ErrInvalidConfig("invalid private key", err)
	}

	prefixedMsg := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(message), message)
	msgHash := crypto.Keccak256Hash([]byte(prefixedMsg))

	sig, err := crypto.Sign(msgHash.Bytes(), key)
	if err != nil {
		return "", web3.ErrInvalidSignature("failed to sign message", err)
	}
	if len(sig) == 65 {
		sig[64] += 27
	}
	return hexutil.Encode(sig), nil
}

// DID represents a Decentralized Identifier (parse/format only — no resolution).
type DID struct {
	Method     string
	Identifier string
	Path       string
	Query      string
	Fragment   string
}

// ParseDID parses a DID string into components. It does not resolve DID documents.
func ParseDID(did string) (*DID, error) {
	// Basic DID regex: did:method:identifier
	re := regexp.MustCompile(`^did:([a-z0-9]+):([a-zA-Z0-9._-]+)(?:/([^?#]*))?(?:\?([^#]*))?(?:#(.*))?$`)
	matches := re.FindStringSubmatch(did)
	if matches == nil {
		return nil, web3.ErrInvalidConfig("invalid DID format", nil)
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

// EthereumDID creates a DID from an Ethereum address (did:ethr:...).
// This does not resolve or verify on-chain identity documents.
func EthereumDID(address string) *DID {
	return &DID{
		Method:     "ethr",
		Identifier: strings.ToLower(address),
	}
}

package pqc

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"io"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"golang.org/x/crypto/hkdf"
)

// Errors (pkg/errors). Dilithium/ML-DSA is not implemented — see package doc.
var (
	ErrInvalidPublicKey    = errors.InvalidArgument("invalid public key", nil)
	ErrInvalidPrivateKey   = errors.InvalidArgument("invalid private key", nil)
	ErrDecapsulationFailed = errors.Internal("decapsulation failed", nil)
	ErrInvalidCiphertext   = errors.InvalidArgument("invalid ciphertext", nil)
)

// =========================================================================
// Hybrid Key Encapsulation
// =========================================================================

// HybridKEM combines classical ECDH with post-quantum Kyber for key exchange.
// This provides "defense in depth" - even if one algorithm is broken,
// the other still provides security.
type HybridKEM struct {
	classicalKEM KEM
	pqKEM        KEM
}

// KEM is the key encapsulation mechanism interface.
type KEM interface {
	// KeyGen generates a key pair.
	KeyGen() (publicKey, privateKey []byte, err error)

	// Encapsulate generates a shared secret and ciphertext.
	Encapsulate(publicKey []byte) (sharedSecret, ciphertext []byte, err error)

	// Decapsulate recovers the shared secret from ciphertext.
	Decapsulate(privateKey, ciphertext []byte) (sharedSecret []byte, err error)

	// PublicKeySize returns the size of public keys.
	PublicKeySize() int

	// CiphertextSize returns the size of ciphertexts.
	CiphertextSize() int

	// SharedSecretSize returns the size of shared secrets.
	SharedSecretSize() int
}

// NewHybridKEM creates a hybrid KEM combining classical and post-quantum.
func NewHybridKEM() *HybridKEM {
	return &HybridKEM{
		classicalKEM: &X25519KEM{},
		pqKEM:        NewKyberKEM(KyberLevel768), // Kyber-768 (NIST Level 3)
	}
}

// KeyGen generates hybrid key pairs.
func (h *HybridKEM) KeyGen() (HybridPublicKey, HybridPrivateKey, error) {
	classicPub, classicPriv, err := h.classicalKEM.KeyGen()
	if err != nil {
		return HybridPublicKey{}, HybridPrivateKey{}, err
	}

	pqPub, pqPriv, err := h.pqKEM.KeyGen()
	if err != nil {
		return HybridPublicKey{}, HybridPrivateKey{}, err
	}

	return HybridPublicKey{
			Classical: classicPub,
			PQ:        pqPub,
		}, HybridPrivateKey{
			Classical: classicPriv,
			PQ:        pqPriv,
		}, nil
}

// Encapsulate creates a shared secret using both KEMs.
func (h *HybridKEM) Encapsulate(pub HybridPublicKey) (sharedSecret []byte, ciphertext HybridCiphertext, err error) {
	// Classical encapsulation
	classicSS, classicCT, err := h.classicalKEM.Encapsulate(pub.Classical)
	if err != nil {
		return nil, HybridCiphertext{}, err
	}

	// Post-quantum encapsulation
	pqSS, pqCT, err := h.pqKEM.Encapsulate(pub.PQ)
	if err != nil {
		return nil, HybridCiphertext{}, err
	}

	// Combine shared secrets using HKDF
	combined := append(classicSS, pqSS...)
	sharedSecret, err = deriveKey(combined, 32)
	if err != nil {
		return nil, HybridCiphertext{}, err
	}

	return sharedSecret, HybridCiphertext{
		Classical: classicCT,
		PQ:        pqCT,
	}, nil
}

// Decapsulate recovers the shared secret.
func (h *HybridKEM) Decapsulate(priv HybridPrivateKey, ct HybridCiphertext) ([]byte, error) {
	// Classical decapsulation
	classicSS, err := h.classicalKEM.Decapsulate(priv.Classical, ct.Classical)
	if err != nil {
		return nil, err
	}

	// Post-quantum decapsulation
	pqSS, err := h.pqKEM.Decapsulate(priv.PQ, ct.PQ)
	if err != nil {
		return nil, err
	}

	// Combine shared secrets
	combined := append(classicSS, pqSS...)
	return deriveKey(combined, 32)
}

// HybridPublicKey contains both classical and PQ public keys.
type HybridPublicKey struct {
	Classical []byte
	PQ        []byte
}

// HybridPrivateKey contains both classical and PQ private keys.
type HybridPrivateKey struct {
	Classical []byte
	PQ        []byte
}

// HybridCiphertext contains both classical and PQ ciphertexts.
type HybridCiphertext struct {
	Classical []byte
	PQ        []byte
}

// =========================================================================
// Hybrid Encryption (KEM + symmetric)
// =========================================================================

// HybridEncryptor provides quantum-resistant encryption.
type HybridEncryptor struct {
	kem *HybridKEM
}

// NewHybridEncryptor creates a new hybrid encryptor.
func NewHybridEncryptor() *HybridEncryptor {
	return &HybridEncryptor{
		kem: NewHybridKEM(),
	}
}

// GenerateKeyPair generates a new hybrid key pair.
func (e *HybridEncryptor) GenerateKeyPair() (HybridPublicKey, HybridPrivateKey, error) {
	return e.kem.KeyGen()
}

// Encrypt encrypts data for the recipient's public key.
func (e *HybridEncryptor) Encrypt(plaintext []byte, recipientPub HybridPublicKey) ([]byte, error) {
	// Generate ephemeral shared secret
	sharedSecret, ciphertext, err := e.kem.Encapsulate(recipientPub)
	if err != nil {
		return nil, err
	}

	// Encrypt data with AES-GCM using shared secret
	block, err := aes.NewCipher(sharedSecret)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	// Format: classical_ct | pq_ct | nonce | encrypted_data
	encryptedData := gcm.Seal(nil, nonce, plaintext, nil)

	result := make([]byte, 0, len(ciphertext.Classical)+len(ciphertext.PQ)+len(nonce)+len(encryptedData)+8)
	result = appendLengthPrefixed(result, ciphertext.Classical)
	result = appendLengthPrefixed(result, ciphertext.PQ)
	result = append(result, nonce...)
	result = append(result, encryptedData...)

	return result, nil
}

// Decrypt decrypts data using the recipient's private key.
func (e *HybridEncryptor) Decrypt(data []byte, recipientPriv HybridPrivateKey) ([]byte, error) {
	// Parse ciphertext
	classicalCT, data, err := readLengthPrefixed(data)
	if err != nil {
		return nil, ErrInvalidCiphertext
	}

	pqCT, data, err := readLengthPrefixed(data)
	if err != nil {
		return nil, ErrInvalidCiphertext
	}

	if len(data) < 12 {
		return nil, ErrInvalidCiphertext
	}

	// Recover shared secret
	sharedSecret, err := e.kem.Decapsulate(recipientPriv, HybridCiphertext{
		Classical: classicalCT,
		PQ:        pqCT,
	})
	if err != nil {
		return nil, err
	}

	// Decrypt data
	block, err := aes.NewCipher(sharedSecret)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := data[:gcm.NonceSize()]
	encryptedData := data[gcm.NonceSize():]

	return gcm.Open(nil, nonce, encryptedData, nil)
}

// =========================================================================
// Helper functions
// =========================================================================

func deriveKey(secret []byte, length int) ([]byte, error) {
	hkdfReader := hkdf.New(sha256.New, secret, nil, []byte("hybrid-kem"))
	key := make([]byte, length)
	if _, err := io.ReadFull(hkdfReader, key); err != nil {
		return nil, err
	}
	return key, nil
}

func appendLengthPrefixed(dst, data []byte) []byte {
	length := uint32(len(data))
	dst = append(dst, byte(length>>24), byte(length>>16), byte(length>>8), byte(length))
	return append(dst, data...)
}

func readLengthPrefixed(data []byte) ([]byte, []byte, error) {
	if len(data) < 4 {
		return nil, nil, ErrInvalidCiphertext
	}
	length := uint32(data[0])<<24 | uint32(data[1])<<16 | uint32(data[2])<<8 | uint32(data[3])
	if len(data) < int(4+length) {
		return nil, nil, ErrInvalidCiphertext
	}
	return data[4 : 4+length], data[4+length:], nil
}

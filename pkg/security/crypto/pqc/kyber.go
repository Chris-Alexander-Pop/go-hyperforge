package pqc

import (
	"github.com/cloudflare/circl/kem"
	"github.com/cloudflare/circl/kem/mlkem/mlkem1024"
	"github.com/cloudflare/circl/kem/mlkem/mlkem512"
	"github.com/cloudflare/circl/kem/mlkem/mlkem768"
)

// KyberLevel selects an ML-KEM (FIPS 203) parameter set.
// Names retain the historical "Kyber" levels; the implementation uses CIRCL ML-KEM.
type KyberLevel int

const (
	// KyberLevel512 provides NIST Level 1 security (ML-KEM-512).
	KyberLevel512 KyberLevel = 512

	// KyberLevel768 provides NIST Level 3 security (ML-KEM-768).
	// Recommended for most applications.
	KyberLevel768 KyberLevel = 768

	// KyberLevel1024 provides NIST Level 5 security (ML-KEM-1024).
	KyberLevel1024 KyberLevel = 1024
)

// KyberKEM implements ML-KEM via Cloudflare CIRCL (FIPS 203).
//
// Despite the historical Kyber naming, KeyGen/Encapsulate/Decapsulate use
// github.com/cloudflare/circl/kem/mlkem. For signatures see DilithiumSigner.
type KyberKEM struct {
	level  KyberLevel
	scheme kem.Scheme
}

// NewKyberKEM creates an ML-KEM at the specified security level.
func NewKyberKEM(level KyberLevel) *KyberKEM {
	return &KyberKEM{
		level:  level,
		scheme: schemeForLevel(level),
	}
}

func schemeForLevel(level KyberLevel) kem.Scheme {
	switch level {
	case KyberLevel512:
		return mlkem512.Scheme()
	case KyberLevel1024:
		return mlkem1024.Scheme()
	default:
		return mlkem768.Scheme()
	}
}

// KeyGen generates an ML-KEM key pair (packed public and private keys).
func (k *KyberKEM) KeyGen() (publicKey, privateKey []byte, err error) {
	pk, sk, err := k.scheme.GenerateKeyPair()
	if err != nil {
		return nil, nil, err
	}
	publicKey, err = pk.MarshalBinary()
	if err != nil {
		return nil, nil, err
	}
	privateKey, err = sk.MarshalBinary()
	if err != nil {
		return nil, nil, err
	}
	return publicKey, privateKey, nil
}

// Encapsulate generates a shared secret and ciphertext for the given public key.
func (k *KyberKEM) Encapsulate(publicKey []byte) (sharedSecret, ciphertext []byte, err error) {
	pk, err := k.scheme.UnmarshalBinaryPublicKey(publicKey)
	if err != nil {
		return nil, nil, ErrInvalidPublicKey
	}
	ct, ss, err := k.scheme.Encapsulate(pk)
	if err != nil {
		return nil, nil, err
	}
	return ss, ct, nil
}

// Decapsulate recovers the shared secret from ciphertext using the private key.
func (k *KyberKEM) Decapsulate(privateKey, ciphertext []byte) (sharedSecret []byte, err error) {
	sk, err := k.scheme.UnmarshalBinaryPrivateKey(privateKey)
	if err != nil {
		return nil, ErrInvalidPrivateKey
	}
	if len(ciphertext) != k.scheme.CiphertextSize() {
		return nil, ErrInvalidCiphertext
	}
	ss, err := k.scheme.Decapsulate(sk, ciphertext)
	if err != nil {
		return nil, ErrDecapsulationFailed
	}
	return ss, nil
}

func (k *KyberKEM) PublicKeySize() int    { return k.scheme.PublicKeySize() }
func (k *KyberKEM) PrivateKeySize() int   { return k.scheme.PrivateKeySize() }
func (k *KyberKEM) CiphertextSize() int   { return k.scheme.CiphertextSize() }
func (k *KyberKEM) SharedSecretSize() int { return k.scheme.SharedKeySize() }

// SchemeName returns the CIRCL scheme name (e.g. "ML-KEM-768").
func (k *KyberKEM) SchemeName() string { return k.scheme.Name() }

// Level returns the configured security level.
func (k *KyberKEM) Level() KyberLevel { return k.level }

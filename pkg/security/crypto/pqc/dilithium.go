package pqc

import (
	"github.com/cloudflare/circl/sign"
	"github.com/cloudflare/circl/sign/mldsa/mldsa44"
	"github.com/cloudflare/circl/sign/mldsa/mldsa65"
	"github.com/cloudflare/circl/sign/mldsa/mldsa87"
)

// DilithiumLevel selects an ML-DSA (FIPS 204) parameter set.
// Names retain the historical "Dilithium" levels; the implementation uses CIRCL ML-DSA.
type DilithiumLevel int

const (
	// DilithiumLevel2 provides NIST Level 2 security (ML-DSA-44).
	DilithiumLevel2 DilithiumLevel = 2

	// DilithiumLevel3 provides NIST Level 3 security (ML-DSA-65).
	// Recommended for most applications.
	DilithiumLevel3 DilithiumLevel = 3

	// DilithiumLevel5 provides NIST Level 5 security (ML-DSA-87).
	DilithiumLevel5 DilithiumLevel = 5
)

// Signer is the post-quantum digital signature interface.
type Signer interface {
	// KeyGen generates a key pair (packed public and private keys).
	KeyGen() (publicKey, privateKey []byte, err error)

	// Sign produces a signature over message using privateKey.
	Sign(privateKey, message []byte) (signature []byte, err error)

	// PublicKeySize returns the size of public keys.
	PublicKeySize() int

	// PrivateKeySize returns the size of private keys.
	PrivateKeySize() int

	// SignatureSize returns the size of signatures.
	SignatureSize() int
}

// Verifier verifies post-quantum signatures.
type Verifier interface {
	// Verify reports whether signature is valid for message under publicKey.
	Verify(publicKey, message, signature []byte) (bool, error)

	// PublicKeySize returns the size of public keys.
	PublicKeySize() int

	// SignatureSize returns the size of signatures.
	SignatureSize() int
}

// DilithiumSigner implements ML-DSA via Cloudflare CIRCL (FIPS 204).
//
// Despite the historical Dilithium naming, KeyGen/Sign/Verify use
// github.com/cloudflare/circl/sign/mldsa (ML-DSA-44/65/87).
type DilithiumSigner struct {
	level  DilithiumLevel
	scheme sign.Scheme
}

// Ensure DilithiumSigner implements Signer and Verifier.
var (
	_ Signer   = (*DilithiumSigner)(nil)
	_ Verifier = (*DilithiumSigner)(nil)
)

// NewDilithiumSigner creates an ML-DSA signer/verifier at the specified security level.
func NewDilithiumSigner(level DilithiumLevel) *DilithiumSigner {
	return &DilithiumSigner{
		level:  level,
		scheme: mldsaSchemeForLevel(level),
	}
}

func mldsaSchemeForLevel(level DilithiumLevel) sign.Scheme {
	switch level {
	case DilithiumLevel2:
		return mldsa44.Scheme()
	case DilithiumLevel5:
		return mldsa87.Scheme()
	default:
		return mldsa65.Scheme()
	}
}

// KeyGen generates an ML-DSA key pair (packed public and private keys).
func (d *DilithiumSigner) KeyGen() (publicKey, privateKey []byte, err error) {
	pk, sk, err := d.scheme.GenerateKey()
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

// Sign produces an ML-DSA signature over message.
func (d *DilithiumSigner) Sign(privateKey, message []byte) ([]byte, error) {
	sk, err := d.scheme.UnmarshalBinaryPrivateKey(privateKey)
	if err != nil {
		return nil, ErrInvalidPrivateKey
	}
	sig := d.scheme.Sign(sk, message, nil)
	if len(sig) == 0 {
		return nil, ErrSignFailed
	}
	return sig, nil
}

// Verify checks an ML-DSA signature.
func (d *DilithiumSigner) Verify(publicKey, message, signature []byte) (bool, error) {
	pk, err := d.scheme.UnmarshalBinaryPublicKey(publicKey)
	if err != nil {
		return false, ErrInvalidPublicKey
	}
	if len(signature) != d.scheme.SignatureSize() {
		return false, ErrInvalidSignature
	}
	return d.scheme.Verify(pk, message, signature, nil), nil
}

func (d *DilithiumSigner) PublicKeySize() int  { return d.scheme.PublicKeySize() }
func (d *DilithiumSigner) PrivateKeySize() int { return d.scheme.PrivateKeySize() }
func (d *DilithiumSigner) SignatureSize() int  { return d.scheme.SignatureSize() }

// SchemeName returns the CIRCL scheme name (e.g. "ML-DSA-65").
func (d *DilithiumSigner) SchemeName() string { return d.scheme.Name() }

// Level returns the configured security level.
func (d *DilithiumSigner) Level() DilithiumLevel { return d.level }

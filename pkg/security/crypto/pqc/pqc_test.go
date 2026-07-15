package pqc_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/security/crypto/pqc"
)

func TestMLKEMRoundTrip(t *testing.T) {
	levels := []pqc.KyberLevel{pqc.KyberLevel512, pqc.KyberLevel768, pqc.KyberLevel1024}
	for _, level := range levels {
		t.Run(fmt.Sprintf("ML-KEM-%d", level), func(t *testing.T) {
			kem := pqc.NewKyberKEM(level)
			pub, priv, err := kem.KeyGen()
			if err != nil {
				t.Fatalf("KeyGen: %v", err)
			}
			if len(pub) != kem.PublicKeySize() || len(priv) != kem.PrivateKeySize() {
				t.Fatalf("unexpected key sizes pub=%d priv=%d", len(pub), len(priv))
			}

			ss1, ct, err := kem.Encapsulate(pub)
			if err != nil {
				t.Fatalf("Encapsulate: %v", err)
			}
			if len(ct) != kem.CiphertextSize() || len(ss1) != kem.SharedSecretSize() {
				t.Fatalf("unexpected encaps sizes ct=%d ss=%d", len(ct), len(ss1))
			}

			ss2, err := kem.Decapsulate(priv, ct)
			if err != nil {
				t.Fatalf("Decapsulate: %v", err)
			}
			if !bytes.Equal(ss1, ss2) {
				t.Fatal("shared secrets do not match")
			}
		})
	}
}

func TestMLKEMInvalidKeys(t *testing.T) {
	kem := pqc.NewKyberKEM(pqc.KyberLevel768)
	if _, _, err := kem.Encapsulate([]byte("short")); err == nil {
		t.Fatal("expected invalid public key")
	}
	pub, priv, err := kem.KeyGen()
	if err != nil {
		t.Fatal(err)
	}
	_, ct, err := kem.Encapsulate(pub)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := kem.Decapsulate([]byte("bad"), ct); err == nil {
		t.Fatal("expected invalid private key")
	}
	if _, err := kem.Decapsulate(priv, []byte("short")); err == nil {
		t.Fatal("expected invalid ciphertext")
	}
}

func TestHybridKEMRoundTrip(t *testing.T) {
	h := pqc.NewHybridKEM()
	pub, priv, err := h.KeyGen()
	if err != nil {
		t.Fatalf("KeyGen: %v", err)
	}
	ss1, ct, err := h.Encapsulate(pub)
	if err != nil {
		t.Fatalf("Encapsulate: %v", err)
	}
	ss2, err := h.Decapsulate(priv, ct)
	if err != nil {
		t.Fatalf("Decapsulate: %v", err)
	}
	if !bytes.Equal(ss1, ss2) {
		t.Fatal("hybrid shared secrets do not match")
	}
}

func TestHybridEncryptor(t *testing.T) {
	e := pqc.NewHybridEncryptor()
	pub, priv, err := e.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	plain := []byte("quantum-resistant payload")
	ct, err := e.Encrypt(plain, pub)
	if err != nil {
		t.Fatal(err)
	}
	out, err := e.Decrypt(ct, priv)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(plain, out) {
		t.Fatalf("got %q want %q", out, plain)
	}
}

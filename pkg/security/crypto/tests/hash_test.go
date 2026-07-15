package tests

import (
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/security/crypto"
)

func TestHasher_Argon2id(t *testing.T) {
	cfg := crypto.DefaultHashConfig()
	// Lower cost for faster tests
	cfg.Argon2Time = 1
	cfg.Argon2Memory = 1024 // 1 MB
	cfg.Argon2Threads = 1

	hasher := crypto.NewHasher(cfg)

	password := "correct-horse-battery-staple"

	hash, err := hasher.Hash(password)
	if err != nil {
		t.Fatalf("Hash failed: %v", err)
	}

	if hash == "" {
		t.Fatal("Hash returned empty string")
	}

	match, err := hasher.Verify(password, hash)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if !match {
		t.Error("Verify failed for correct password")
	}

	match, err = hasher.Verify("wrong-password", hash)
	if err != nil {
		t.Fatalf("Verify failed (wrong pw): %v", err)
	}
	if match {
		t.Error("Verify succeeded for wrong password")
	}
}

func TestHasher_Bcrypt(t *testing.T) {
	cfg := crypto.DefaultHashConfig()
	cfg.Algorithm = "bcrypt"
	cfg.BcryptCost = 4 // Minimum cost for speed

	hasher := crypto.NewHasher(cfg)

	password := "my-secret-password"

	hash, err := hasher.Hash(password)
	if err != nil {
		t.Fatalf("Hash failed: %v", err)
	}

	match, err := hasher.Verify(password, hash)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if !match {
		t.Error("Verify failed for correct password")
	}
}

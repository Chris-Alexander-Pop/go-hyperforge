package otp

import (
	"crypto/sha256"
	"fmt"
	"testing"
)

func TestHashRecoveryCode(t *testing.T) {
	code := "ABCD-EFGH"
	expectedHash := sha256.Sum256([]byte("abcdefgh"))
	expectedHex := fmt.Sprintf("%x", expectedHash)

	hashed := HashRecoveryCode(code)

	if hashed != expectedHex {
		t.Errorf("HashRecoveryCode(%q) = %q; want %q", code, hashed, expectedHex)
	}
}

func TestRecoveryCodeManager_GenerateCodes(t *testing.T) {
	manager := NewRecoveryCodeManager(RecoveryCodeConfig{
		Count:     5,
		Length:    10,
		GroupSize: 5,
	})

	displayCodes, hashedCodes, err := manager.GenerateCodes()
	if err != nil {
		t.Fatalf("GenerateCodes() error = %v", err)
	}

	if len(displayCodes) != 5 {
		t.Errorf("len(displayCodes) = %d; want 5", len(displayCodes))
	}
	if len(hashedCodes) != 5 {
		t.Errorf("len(hashedCodes) = %d; want 5", len(hashedCodes))
	}

	// Verify hashedCodes are not plaintext
	for i, raw := range displayCodes {
		hashed := hashedCodes[i]
		if hashed == raw {
			t.Errorf("hashedCodes[%d] is plaintext: %q", i, raw)
		}

		// Verify consistent hashing
		computedHash := HashRecoveryCode(raw)
		if hashed != computedHash {
			t.Errorf("hashedCodes[%d] = %q; want %q (from raw %q)", i, hashed, computedHash, raw)
		}
	}
}

func TestRecoveryCodeSet_Validate(t *testing.T) {
	codeRaw := "1234567890"
	codeHashed := HashRecoveryCode(codeRaw)

	// Create set with manually hashed code
	set := NewRecoveryCodeSet([]string{codeHashed})

	// Test valid code
	if valid := set.Validate(codeRaw); !valid {
		t.Errorf("Validate(%q) = false; want true", codeRaw)
	}

	// Test reuse (should be false)
	if valid := set.Validate(codeRaw); valid {
		t.Errorf("Validate(%q) = true; want false (already used)", codeRaw)
	}

	// Test invalid code
	if valid := set.Validate("invalid"); valid {
		t.Errorf("Validate(%q) = true; want false", "invalid")
	}
}

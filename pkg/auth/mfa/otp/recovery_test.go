package otp

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

func TestRecoveryCodesAreHashed(t *testing.T) {
	manager := NewRecoveryCodeManager(DefaultRecoveryCodeConfig())
	displayCodes, hashedCodes, err := manager.GenerateCodes()
	if err != nil {
		t.Fatalf("Failed to generate codes: %v", err)
	}

	if len(displayCodes) != len(hashedCodes) {
		t.Fatalf("Mismatch in codes length")
	}

	for i, display := range displayCodes {
		// Normalize display code (remove dashes, lowercase)
		normalized := strings.ReplaceAll(strings.ToLower(display), "-", "")

		// CURRENT BEHAVIOR (VULNERABLE): hashedCode is just the normalized code
		// DESIRED BEHAVIOR (SECURE): hashedCode is sha256(normalized)

		// Let's assert that it IS hashed (this will fail currently)
		hash := sha256.Sum256([]byte(normalized))
		expectedHash := hex.EncodeToString(hash[:])

		if hashedCodes[i] != expectedHash {
			t.Errorf("Code index %d: stored hash '%s' does not match expected sha256 hash '%s'. Seems like it is stored in plaintext!", i, hashedCodes[i], expectedHash)
		}
	}
}

func TestRecoveryCodeSet_Validate(t *testing.T) {
	// Simulate hashed codes (as they should be)
	rawCode := "abcd1234efgh5678"
	hash := sha256.Sum256([]byte(rawCode))
	hashedCode := hex.EncodeToString(hash[:])

	set := NewRecoveryCodeSet([]string{hashedCode})

	// Validate should hash the input and check against the set
	if !set.Validate(rawCode) {
		t.Errorf("Validation failed for valid code")
	}

	if set.Validate("wrongcode") {
		t.Errorf("Validation succeeded for invalid code")
	}
}

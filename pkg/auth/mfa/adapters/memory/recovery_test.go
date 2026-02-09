package memory

import (
	"context"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/auth/mfa"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecoveryCodeHashing(t *testing.T) {
	ctx := context.Background()
	p := New(mfa.Config{
		TOTPIssuer: "Test",
		TOTPDigits: 6,
		TOTPPeriod: 30,
	})

	userID := "user-123"
	_, displayCodes, err := p.Enroll(ctx, userID)
	require.NoError(t, err)
	require.NotEmpty(t, displayCodes)

	// Access private field enrollments to check stored codes
	p.mu.RLock()
	enrollment, ok := p.enrollments[userID]
	p.mu.RUnlock()
	require.True(t, ok)

	// Check if codes are hashed (SHA-256 hex string is 64 chars)
	// Before fix: This is expected to fail (length is 16)
	if len(enrollment.Recovery) > 0 {
		// We expect 64 chars for SHA-256 hash.
		// If it's 16, it means it's just the raw hex of 8 bytes.
		assert.Equal(t, 64, len(enrollment.Recovery[0]), "Recovery codes should be SHA-256 hashed (64 hex chars)")
	}

	// Verify recovery works with one of the display codes (which contain dashes)
	code := displayCodes[0]

	// Manually enable enrollment to test recovery
	p.mu.Lock()
	enrollment.Enabled = true
	p.mu.Unlock()

	// Try to recover
	// Before fix: This might fail if the implementation doesn't handle dashes or hashing correctly
	success, err := p.Recover(ctx, userID, code)
	require.NoError(t, err)
	assert.True(t, success, "Recovery should succeed with valid display code")
}

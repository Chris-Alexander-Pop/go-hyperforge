package redis

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/chris-alexander-pop/system-design-library/pkg/auth/mfa"
	"github.com/chris-alexander-pop/system-design-library/pkg/auth/mfa/otp"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)


func TestRecoveryCodes(t *testing.T) {
	ctx := context.Background()
	s := miniredis.RunT(t)
	defer s.Close()

	client := redis.NewClient(&redis.Options{Addr: s.Addr()})
	defer client.Close()

	config := mfa.Config{TOTPIssuer: "TestApp", TOTPDigits: 6, TOTPPeriod: 30}
	provider := New(client, config)

	userID := "user-recovery-test"

	// 1. Enroll
	secret, recoveryCodes, err := provider.Enroll(ctx, userID)
	require.NoError(t, err)
	require.NotEmpty(t, recoveryCodes)

	// 2. Complete Enrollment
	totpGen := otp.NewTOTP(otp.TOTPConfig{Issuer: config.TOTPIssuer, Digits: config.TOTPDigits, Period: config.TOTPPeriod})
	code, err := totpGen.GenerateCode(secret)
	require.NoError(t, err)
	err = provider.CompleteEnrollment(ctx, userID, code)
	require.NoError(t, err)

	// 3. Recover
	recoveryCode := recoveryCodes[0]
	success, err := provider.Recover(ctx, userID, recoveryCode)
	require.NoError(t, err)
	assert.True(t, success, "Recovery should succeed")

	// 4. Try to reuse recovery code (should fail if removed)
	success, err = provider.Recover(ctx, userID, recoveryCode)
	// Actually Recover returns false if not found, or error?
	// Implementation: returns success=false, err=nil if code invalid/not found.
	require.NoError(t, err)
	assert.False(t, success, "Recovery code should be consumed")
}

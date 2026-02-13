package redis_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/chris-alexander-pop/system-design-library/pkg/auth/mfa"
	redisAdapter "github.com/chris-alexander-pop/system-design-library/pkg/auth/mfa/adapters/redis"
	"github.com/chris-alexander-pop/system-design-library/pkg/auth/mfa/otp"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVerifyReplayProtection(t *testing.T) {
	ctx := context.Background()

	// Start Miniredis
	s, err := miniredis.Run()
	require.NoError(t, err)
	defer s.Close()

	client := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})
	defer client.Close()

	// Setup MFA Provider
	config := mfa.Config{
		TOTPIssuer: "TestApp",
		TOTPDigits: 6,
		TOTPPeriod: 30, // 30 seconds
	}
	provider := redisAdapter.New(client, config)

	userID := "user-replay-test"

	// 1. Enroll
	secret, _, err := provider.Enroll(ctx, userID)
	require.NoError(t, err)

	// 2. Complete Enrollment
	totp := otp.NewTOTP(otp.TOTPConfig{
		Issuer: config.TOTPIssuer,
		Digits: config.TOTPDigits,
		Period: config.TOTPPeriod,
	})

	code1, err := totp.GenerateCode(secret)
	require.NoError(t, err)

	err = provider.CompleteEnrollment(ctx, userID, code1)
	require.NoError(t, err)

	// 3. Verify (First time)
	// Note: We use a different code (code2) because code1 was already used for enrollment.
	// Since Skew=1, a code for the next period is also valid now.
	code2, err := totp.GenerateCodeAt(secret, time.Now().Add(time.Duration(config.TOTPPeriod)*time.Second))
	require.NoError(t, err)

	valid, err := provider.Verify(ctx, userID, code2)
	assert.NoError(t, err)
	assert.True(t, valid, "First verification with fresh code should succeed")

	// 4. Verify (Second time) - Replay
	valid, err = provider.Verify(ctx, userID, code2)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "code already used")
	assert.False(t, valid, "Second verification should fail")
}

func TestEnrollmentReplayProtection(t *testing.T) {
	ctx := context.Background()

	// Start Miniredis
	s, err := miniredis.Run()
	require.NoError(t, err)
	defer s.Close()

	client := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})
	defer client.Close()

	// Setup MFA Provider
	config := mfa.Config{
		TOTPIssuer: "TestApp",
		TOTPDigits: 6,
		TOTPPeriod: 30, // 30 seconds
	}
	provider := redisAdapter.New(client, config)

	userID := "user-enrollment-replay-test"

	// 1. Enroll
	secret, _, err := provider.Enroll(ctx, userID)
	require.NoError(t, err)

	// 2. Generate a valid code
	totp := otp.NewTOTP(otp.TOTPConfig{
		Issuer: config.TOTPIssuer,
		Digits: config.TOTPDigits,
		Period: config.TOTPPeriod,
	})

	code1, err := totp.GenerateCode(secret)
	require.NoError(t, err)

	// 3. Complete Enrollment with the code
	err = provider.CompleteEnrollment(ctx, userID, code1)
	require.NoError(t, err)

	// 4. Try to Verify with the SAME code immediately
	// This should FAIL if replay protection covers enrollment.
	valid, err := provider.Verify(ctx, userID, code1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "code already used")
	assert.False(t, valid, "Verification should fail for replayed code")
}

func TestRecoveryCodes(t *testing.T) {
	ctx := context.Background()

	// Start Miniredis
	s, err := miniredis.Run()
	require.NoError(t, err)
	defer s.Close()

	client := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})
	defer client.Close()

	// Setup MFA Provider
	config := mfa.Config{
		TOTPIssuer: "TestApp",
		TOTPDigits: 6,
		TOTPPeriod: 30, // 30 seconds
	}
	provider := redisAdapter.New(client, config)

	userID := "user-recovery-test"

	// 1. Enroll
	secret, recoveryCodes, err := provider.Enroll(ctx, userID)
	require.NoError(t, err)
	require.Len(t, recoveryCodes, 10)

	// 2. Complete Enrollment
	totp := otp.NewTOTP(otp.TOTPConfig{
		Issuer: config.TOTPIssuer,
		Digits: config.TOTPDigits,
		Period: config.TOTPPeriod,
	})
	code, err := totp.GenerateCode(secret)
	require.NoError(t, err)
	err = provider.CompleteEnrollment(ctx, userID, code)
	require.NoError(t, err)

	// 3. Recover with a valid code
	// Recovery codes are formatted like "abcd-efgh...". We need to normalize?
	// The provider returns formatted codes.
	// The Normalize function removes dashes and lowercases.
	// Let's assume the user enters the code AS DISPLAYED (with dashes) or without.
	// We'll test with the exact string returned by Enroll.

	validCode := recoveryCodes[0]
	// Remove formatting to simulate user input if they typed it without dashes?
	// Actually, our Recover logic checks hash against stored hash.
	// HashRecoveryCode normalizes input.
	// So input can be "AAAA-BBBB" or "aaaabbbb".

	success, err := provider.Recover(ctx, userID, validCode)
	require.NoError(t, err)
	assert.True(t, success, "Recovery with valid code should succeed")

	// 4. Recover with the same code again (should fail)
	success, err = provider.Recover(ctx, userID, validCode)
	require.NoError(t, err)
	assert.False(t, success, "Recovery with used code should fail")

	// 5. Recover with invalid code
	success, err = provider.Recover(ctx, userID, "invalid-code")
	require.NoError(t, err)
	assert.False(t, success, "Recovery with invalid code should fail")
}

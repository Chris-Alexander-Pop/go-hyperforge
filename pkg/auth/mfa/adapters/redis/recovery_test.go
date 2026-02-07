package redis_test

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/chris-alexander-pop/system-design-library/pkg/auth/mfa"
	redisAdapter "github.com/chris-alexander-pop/system-design-library/pkg/auth/mfa/adapters/redis"
	"github.com/chris-alexander-pop/system-design-library/pkg/auth/mfa/otp"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecoveryFlow(t *testing.T) {
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
		TOTPPeriod: 30,
	}
	provider := redisAdapter.New(client, config)

	userID := "user-recovery-test"

	// 1. Enroll
	secret, recoveryCodes, err := provider.Enroll(ctx, userID)
	require.NoError(t, err)
	require.NotEmpty(t, recoveryCodes)

	// 2. Complete Enrollment (using TOTP)
	totp := otp.NewTOTP(otp.TOTPConfig{
		Issuer: config.TOTPIssuer,
		Digits: config.TOTPDigits,
		Period: config.TOTPPeriod,
	})
	code, err := totp.GenerateCode(secret)
	require.NoError(t, err)

	err = provider.CompleteEnrollment(ctx, userID, code)
	require.NoError(t, err)

	// 3. Recover with Valid Code
	// Use the first recovery code
	recoveryCode := recoveryCodes[0]

	success, err := provider.Recover(ctx, userID, recoveryCode)
	assert.NoError(t, err)
	assert.True(t, success, "Recovery with valid code should succeed")

	// 4. Recover with Same Code (Replay)
	success, err = provider.Recover(ctx, userID, recoveryCode)
	assert.NoError(t, err)
	assert.False(t, success, "Recovery with used code should fail")

	// 5. Recover with Invalid Code
	success, err = provider.Recover(ctx, userID, "invalid-code")
	assert.NoError(t, err)
	assert.False(t, success, "Recovery with invalid code should fail")
}

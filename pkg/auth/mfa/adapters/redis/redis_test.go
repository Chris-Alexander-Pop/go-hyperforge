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
	valid, err := provider.Verify(ctx, userID, code1)
	assert.NoError(t, err)
	assert.True(t, valid, "First verification should succeed")

	// 4. Verify (Second time) - Replay
	valid, err = provider.Verify(ctx, userID, code1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "code already used")
	assert.False(t, valid, "Second verification should fail")

	// 5. Verify different code
	code2, err := totp.GenerateCodeAt(secret, time.Now().Add(time.Duration(config.TOTPPeriod)*time.Second))
	require.NoError(t, err)

	if code2 != code1 {
		valid, err = provider.Verify(ctx, userID, code2)
		assert.NoError(t, err)
		assert.True(t, valid, "Different code should succeed")
	}
}

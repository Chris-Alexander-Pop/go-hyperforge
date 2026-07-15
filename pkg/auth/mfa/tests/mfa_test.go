package tests

import (
	"testing"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/auth/mfa"
	"github.com/chris-alexander-pop/system-design-library/pkg/auth/mfa/adapters/memory"
	"github.com/chris-alexander-pop/system-design-library/pkg/auth/mfa/otp"
	"github.com/chris-alexander-pop/system-design-library/pkg/test"
)

type MFATestSuite struct {
	test.Suite
	provider mfa.Provider
}

func (s *MFATestSuite) SetupTest() {
	s.Suite.SetupTest()
	p, err := memory.New(mfa.Config{
		TOTPIssuer: "TestApp",
		TOTPDigits: 6,
		TOTPPeriod: 30,
	})
	s.Require().NoError(err)
	s.provider = p
}

func (s *MFATestSuite) TestEnrollmentFlow() {
	userID := "user-123"

	secret, codes, err := s.provider.Enroll(s.Ctx, userID)
	s.NoError(err)
	s.NotEmpty(secret)
	s.NotEmpty(codes)

	valid, err := s.provider.Verify(s.Ctx, userID, "123456")
	s.Error(err) // Forbidden (not enabled)
	s.False(valid)

	totp := otp.NewTOTP(otp.TOTPConfig{Issuer: "TestApp", Digits: 6, Period: 30})
	code, err := totp.GenerateCode(secret)
	s.NoError(err)
	s.NoError(s.provider.CompleteEnrollment(s.Ctx, userID, code))

	// Use next period code to avoid enrollment replay collision.
	code2, err := totp.GenerateCodeAt(secret, time.Now().Add(30*time.Second))
	s.NoError(err)
	ok, err := s.provider.Verify(s.Ctx, userID, code2)
	s.NoError(err)
	s.True(ok)
}

func (s *MFATestSuite) TestEncryptedSecretRoundTrip() {
	p, err := memory.New(mfa.Config{
		TOTPIssuer:    "TestApp",
		TOTPDigits:    6,
		TOTPPeriod:    30,
		EncryptionKey: "dev-mfa-encryption-passphrase",
	})
	s.Require().NoError(err)

	secret, _, err := p.Enroll(s.Ctx, "enc-user")
	s.NoError(err)

	totp := otp.NewTOTP(otp.TOTPConfig{Issuer: "TestApp", Digits: 6, Period: 30})
	code, err := totp.GenerateCode(secret)
	s.NoError(err)
	s.NoError(p.CompleteEnrollment(s.Ctx, "enc-user", code))

	code2, err := totp.GenerateCodeAt(secret, time.Now().Add(30*time.Second))
	s.NoError(err)
	ok, err := p.Verify(s.Ctx, "enc-user", code2)
	s.NoError(err)
	s.True(ok)
}

func (s *MFATestSuite) TestDisable() {
	userID := "user-456"
	_, _, err := s.provider.Enroll(s.Ctx, userID)
	s.NoError(err)

	err = s.provider.Disable(s.Ctx, userID)
	s.NoError(err)

	err = s.provider.Disable(s.Ctx, userID)
	s.Error(err) // NotFound
}

func TestMFASuite(t *testing.T) {
	test.Run(t, new(MFATestSuite))
}

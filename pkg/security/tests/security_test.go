package tests

import (
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/security"
	"github.com/chris-alexander-pop/system-design-library/pkg/test"
)

type SecurityRootSuite struct {
	test.Suite
}

func (s *SecurityRootSuite) TestProviderConstants() {
	s.Equal("memory", security.ProviderMemory)
	s.Equal("recaptcha", security.ProviderRecaptcha)
	s.Equal("vault", security.ProviderVault)
	s.Equal("aws-kms", security.ProviderAWSKMS)
}

func (s *SecurityRootSuite) TestSentinelErrors() {
	s.NotNil(security.ErrNotFound)
	s.NotNil(security.ErrNotSupported)
	s.Contains(security.ErrInvalid("bad input", nil).Error(), "bad input")
}

func TestSecurityRootSuite(t *testing.T) {
	test.Run(t, new(SecurityRootSuite))
}

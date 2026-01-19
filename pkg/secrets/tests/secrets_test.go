package tests

import (
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/secrets"
	"github.com/chris-alexander-pop/system-design-library/pkg/secrets/adapters/memory"
	"github.com/chris-alexander-pop/system-design-library/pkg/test"
)

type SecretsTestSuite struct {
	test.Suite
	manager secrets.Manager
}

func (s *SecretsTestSuite) SetupTest() {
	s.Suite.SetupTest()
	s.manager = memory.New()
}

func (s *SecretsTestSuite) TearDownTest() {
	s.manager.Close()
}

func (s *SecretsTestSuite) TestGetSetDelete() {
	key := "api-key"
	val := "12345"

	// Get missing
	_, err := s.manager.GetSecret(s.Ctx, key)
	s.Error(err)

	// Set
	err = s.manager.SetSecret(s.Ctx, key, val)
	s.NoError(err)

	// Get existing
	retrieved, err := s.manager.GetSecret(s.Ctx, key)
	s.NoError(err)
	s.Equal(val, retrieved)

	// Delete
	err = s.manager.DeleteSecret(s.Ctx, key)
	s.NoError(err)

	// Get deleted
	_, err = s.manager.GetSecret(s.Ctx, key)
	s.Error(err)
}

func TestSecretsSuite(t *testing.T) {
	test.Run(t, new(SecretsTestSuite))
}

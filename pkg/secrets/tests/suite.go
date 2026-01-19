package tests

import (
	"github.com/chris-alexander-pop/system-design-library/pkg/secrets"
	"github.com/chris-alexander-pop/system-design-library/pkg/test"
)

type SecretsTestSuite struct {
	test.Suite
	Manager secrets.Manager
}

func (s *SecretsTestSuite) SetupTest() {
	s.Suite.SetupTest()
	// Manager must be injected by embedding suite or initializing before running
}

func (s *SecretsTestSuite) TearDownTest() {
	if s.Manager != nil {
		s.Manager.Close()
	}
}

func (s *SecretsTestSuite) TestGetSetDelete() {
	key := "api-key"
	val := "12345"

	// Get missing
	_, err := s.Manager.GetSecret(s.Ctx, key)
	s.Error(err)

	// Set
	err = s.Manager.SetSecret(s.Ctx, key, val)
	s.NoError(err)

	// Get existing
	retrieved, err := s.Manager.GetSecret(s.Ctx, key)
	s.NoError(err)
	s.Equal(val, retrieved)

	// Delete
	err = s.Manager.DeleteSecret(s.Ctx, key)
	s.NoError(err)

	// Get deleted
	_, err = s.Manager.GetSecret(s.Ctx, key)
	s.Error(err)
}

package memory_test

import (
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/secrets/adapters/memory"
	"github.com/chris-alexander-pop/system-design-library/pkg/secrets/tests"
	"github.com/chris-alexander-pop/system-design-library/pkg/test"
)

type MemorySecretsTestSuite struct {
	tests.SecretsTestSuite
}

func (s *MemorySecretsTestSuite) SetupTest() {
	s.SecretsTestSuite.SetupTest()
	s.Manager = memory.New()
}

func TestMemorySecrets(t *testing.T) {
	test.Run(t, new(MemorySecretsTestSuite))
}

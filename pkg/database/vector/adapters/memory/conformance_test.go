package memory_test

import (
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/database/vector/adapters/memory"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/database/vector/testsuite"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/test"
)

type MemorySuite struct {
	testsuite.VectorSuite
}

func (s *MemorySuite) SetupTest() {
	s.Suite.SetupTest()
	store := memory.New()
	s.Store = store
	s.Cleanup = func() {
		_ = store.Close()
	}
}

func TestMemoryVectorConformance(t *testing.T) {
	test.Run(t, &MemorySuite{VectorSuite: testsuite.VectorSuite{Suite: test.NewSuite()}})
}

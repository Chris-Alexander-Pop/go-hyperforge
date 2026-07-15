package memory_test

import (
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/database/graph/adapters/memory"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/database/graph/testsuite"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/test"
)

type MemorySuite struct {
	testsuite.GraphSuite
}

func (s *MemorySuite) SetupTest() {
	s.Suite.SetupTest()
	store := memory.New()
	s.Store = store
	s.Cleanup = func() {
		_ = store.Close()
	}
}

func TestMemoryGraph(t *testing.T) {
	test.Run(t, &MemorySuite{GraphSuite: testsuite.GraphSuite{Suite: test.NewSuite()}})
}

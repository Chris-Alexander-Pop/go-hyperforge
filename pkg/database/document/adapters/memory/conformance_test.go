package memory_test

import (
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/database/document/adapters/memory"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/database/document/testsuite"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/test"
)

type MemorySuite struct {
	testsuite.DocumentSuite
}

func (s *MemorySuite) SetupTest() {
	s.Suite.SetupTest()
	store := memory.New()
	s.Store = store
	s.Cleanup = func() {
		_ = store.Close()
	}
}

func TestMemoryDocumentConformance(t *testing.T) {
	test.Run(t, &MemorySuite{DocumentSuite: testsuite.DocumentSuite{Suite: test.NewSuite()}})
}

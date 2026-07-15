package memory_test

import (
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/database/kv/adapters/memory"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/database/kv/testsuite"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/test"
)

type MemorySuite struct {
	testsuite.KVSuite
}

func (s *MemorySuite) SetupTest() {
	s.Suite.SetupTest()
	store := memory.New()
	s.Store = store
	s.Cleanup = func() {
		_ = store.Close()
	}
}

func TestMemoryKV(t *testing.T) {
	test.Run(t, &MemorySuite{KVSuite: testsuite.KVSuite{Suite: test.NewSuite()}})
}

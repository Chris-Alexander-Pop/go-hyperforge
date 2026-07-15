package memory_test

import (
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/storage/blob"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/storage/blob/adapters/memory"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/storage/blob/testsuite"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/test"
)

type MemorySuite struct {
	testsuite.BlobSuite
}

func (s *MemorySuite) SetupTest() {
	s.Suite.SetupTest()
	s.Store = memory.New(blob.Config{})
}

func TestMemoryBlob(t *testing.T) {
	test.Run(t, &MemorySuite{BlobSuite: testsuite.BlobSuite{Suite: test.NewSuite()}})
}

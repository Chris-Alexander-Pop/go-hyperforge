package memory_test

import (
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/storage/blob"
	"github.com/chris-alexander-pop/system-design-library/pkg/storage/blob/adapters/memory"
	"github.com/chris-alexander-pop/system-design-library/pkg/storage/blob/tests"
	"github.com/chris-alexander-pop/system-design-library/pkg/test"
)

type MemorySuite struct {
	tests.BlobSuite
}

func (s *MemorySuite) SetupTest() {
	s.Suite.SetupTest()
	s.Store = memory.New(blob.Config{})
}

func TestMemoryBlob(t *testing.T) {
	test.Run(t, &MemorySuite{BlobSuite: tests.BlobSuite{Suite: test.NewSuite()}})
}

package local_test

import (
	"os"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/storage/blob"
	"github.com/chris-alexander-pop/system-design-library/pkg/storage/blob/adapters/local"
	"github.com/chris-alexander-pop/system-design-library/pkg/storage/blob/tests"
	"github.com/chris-alexander-pop/system-design-library/pkg/test"
)

type LocalSuite struct {
	tests.BlobSuite
}

func (s *LocalSuite) SetupTest() {
	s.Suite.SetupTest()
	dir, err := os.MkdirTemp("", "blob-test-local-*")
	s.Require().NoError(err)

	store, err := local.New(blob.Config{LocalDir: dir})
	s.Require().NoError(err)

	s.Store = store
	s.Cleanup = func() {
		os.RemoveAll(dir)
	}
}

func TestLocalBlob(t *testing.T) {
	test.Run(t, &LocalSuite{BlobSuite: tests.BlobSuite{Suite: test.NewSuite()}})
}

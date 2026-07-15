package local_test

import (
	"os"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/storage/blob"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/storage/blob/adapters/local"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/storage/blob/testsuite"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/test"
)

type LocalSuite struct {
	testsuite.BlobSuite
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
	test.Run(t, &LocalSuite{BlobSuite: testsuite.BlobSuite{Suite: test.NewSuite()}})
}

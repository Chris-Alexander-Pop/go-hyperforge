package tests

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/blob"
	"github.com/chris-alexander-pop/system-design-library/pkg/blob/adapters/local"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"github.com/chris-alexander-pop/system-design-library/pkg/test"
)

type BlobSuite struct {
	*test.Suite
	store blob.Store
	dir   string
}

func TestBlobSuite(t *testing.T) {
	test.Run(t, &BlobSuite{Suite: test.NewSuite()})
}

func (s *BlobSuite) SetupTest() {
	s.Suite.SetupTest()
	// Create temp dir for testing
	dir, err := os.MkdirTemp("", "blob-test-*")
	s.Require().NoError(err)
	s.dir = dir

	localStore, err := local.New(blob.Config{LocalDir: dir})
	s.Require().NoError(err)

	// Use the instrumented store to test it concurrently
	s.store = blob.NewInstrumentedStore(localStore, "test-blob")
}

func (s *BlobSuite) TearDownTest() {
	os.RemoveAll(s.dir)
}

func (s *BlobSuite) TestUploadDownloadDelete() {
	ctx := context.Background()
	key := "folder/test.txt"
	content := "hello world"

	// Upload
	err := s.store.Upload(ctx, key, strings.NewReader(content))
	s.NoError(err)

	// Download
	rc, err := s.store.Download(ctx, key)
	s.NoError(err)
	defer rc.Close()

	readContent, err := io.ReadAll(rc)
	s.NoError(err)
	s.Equal(content, string(readContent))

	// Delete
	err = s.store.Delete(ctx, key)
	s.NoError(err)

	// Verify Gone
	_, err = s.store.Download(ctx, key)
	s.Error(err)

	// Check specific error code
	var appErr *errors.AppError
	if errors.As(err, &appErr) {
		s.Equal(errors.CodeNotFound, appErr.Code)
	} else {
		s.Fail("expected AppError")
	}
}

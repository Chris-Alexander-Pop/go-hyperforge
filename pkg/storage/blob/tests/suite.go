package tests

import (
	"context"
	"io"
	"strings"

	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"github.com/chris-alexander-pop/system-design-library/pkg/storage/blob"
	"github.com/chris-alexander-pop/system-design-library/pkg/test"
)

type BlobSuite struct {
	*test.Suite
	Store blob.Store
	// Optional cleanup
	Cleanup func()
}

// Remove TestBlobSuite as it shouldn't run independently without store

func (s *BlobSuite) TearDownTest() {
	if s.Cleanup != nil {
		s.Cleanup()
	}
}

func (s *BlobSuite) TestUploadDownloadDelete() {
	ctx := context.Background()
	key := "folder/test.txt"
	content := "hello world"

	// Upload
	err := s.Store.Upload(ctx, key, strings.NewReader(content))
	s.NoError(err)

	// Download
	rc, err := s.Store.Download(ctx, key)
	s.NoError(err)
	defer rc.Close()

	readContent, err := io.ReadAll(rc)
	s.NoError(err)
	s.Equal(content, string(readContent))

	// Delete
	err = s.Store.Delete(ctx, key)
	s.NoError(err)

	// Verify Gone
	_, err = s.Store.Download(ctx, key)
	s.Error(err)

	// Check specific error code
	var appErr *errors.AppError
	if errors.As(err, &appErr) {
		s.Equal(errors.CodeNotFound, appErr.Code)
	} else {
		s.Fail("expected AppError")
	}
}

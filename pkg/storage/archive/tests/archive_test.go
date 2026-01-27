package tests

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/storage/archive"
	"github.com/chris-alexander-pop/system-design-library/pkg/storage/archive/adapters/memory"
	"github.com/stretchr/testify/suite"
)

// ArchiveStoreSuite provides a generic test suite for ArchiveStore implementations.
type ArchiveStoreSuite struct {
	suite.Suite
	store archive.ArchiveStore
	ctx   context.Context
}

// SetupTest runs before each test.
func (s *ArchiveStoreSuite) SetupTest() {
	s.store = memory.New()
	s.ctx = context.Background()
}

func (s *ArchiveStoreSuite) TestArchive() {
	data := []byte("backup data content here")
	err := s.store.Archive(s.ctx, "backups/db-2024.sql.gz", bytes.NewReader(data), archive.ArchiveOptions{
		StorageClass: archive.StorageClassArchive,
		Metadata:     map[string]string{"source": "database"},
	})
	s.Require().NoError(err)

	obj, err := s.store.GetObject(s.ctx, "backups/db-2024.sql.gz")
	s.Require().NoError(err)
	s.Equal("backups/db-2024.sql.gz", obj.Key)
	s.Equal(int64(len(data)), obj.Size)
	s.Equal(archive.StorageClassArchive, obj.StorageClass)
	s.NotEmpty(obj.Checksum)
}

func (s *ArchiveStoreSuite) TestArchiveWithMetadata() {
	data := []byte("data with metadata")
	err := s.store.Archive(s.ctx, "with-meta.txt", bytes.NewReader(data), archive.ArchiveOptions{
		Metadata:    map[string]string{"key": "value", "env": "prod"},
		ContentType: "text/plain",
	})
	s.Require().NoError(err)

	obj, err := s.store.GetObject(s.ctx, "with-meta.txt")
	s.Require().NoError(err)
	s.Equal("value", obj.Metadata["key"])
	s.Equal("prod", obj.Metadata["env"])
}

func (s *ArchiveStoreSuite) TestRestoreAndDownload() {
	// Archive first
	data := []byte("data to restore")
	err := s.store.Archive(s.ctx, "restore-me.txt", bytes.NewReader(data), archive.ArchiveOptions{})
	s.Require().NoError(err)

	// Before restore, download should fail
	_, err = s.store.Download(s.ctx, "restore-me.txt")
	s.Error(err)

	// Initiate restore
	job, err := s.store.Restore(s.ctx, "restore-me.txt", archive.RestoreOptions{
		Tier: archive.RestoreTierStandard,
		TTL:  24 * time.Hour,
	})
	s.Require().NoError(err)
	s.NotEmpty(job.ID)
	s.Equal("restore-me.txt", job.Key)
	s.Equal(archive.RestoreStatusCompleted, job.Status) // Memory adapter restores instantly

	// Now download should work
	rc, err := s.store.Download(s.ctx, "restore-me.txt")
	s.Require().NoError(err)
	defer rc.Close()

	downloaded, err := io.ReadAll(rc)
	s.Require().NoError(err)
	s.Equal(data, downloaded)
}

func (s *ArchiveStoreSuite) TestGetRestoreStatus() {
	data := []byte("status check data")
	err := s.store.Archive(s.ctx, "status-check.txt", bytes.NewReader(data), archive.ArchiveOptions{})
	s.Require().NoError(err)

	// No restore job initially
	_, err = s.store.GetRestoreStatus(s.ctx, "status-check.txt")
	s.Error(err)

	// Initiate restore
	_, err = s.store.Restore(s.ctx, "status-check.txt", archive.RestoreOptions{})
	s.Require().NoError(err)

	// Now should have status
	job, err := s.store.GetRestoreStatus(s.ctx, "status-check.txt")
	s.Require().NoError(err)
	s.Equal(archive.RestoreStatusCompleted, job.Status)
}

func (s *ArchiveStoreSuite) TestDownloadNotFound() {
	_, err := s.store.Download(s.ctx, "nonexistent")
	s.Error(err)
}

func (s *ArchiveStoreSuite) TestDownloadNotRestored() {
	data := []byte("not restored yet")
	err := s.store.Archive(s.ctx, "not-restored.txt", bytes.NewReader(data), archive.ArchiveOptions{})
	s.Require().NoError(err)

	_, err = s.store.Download(s.ctx, "not-restored.txt")
	s.Error(err)
}

func (s *ArchiveStoreSuite) TestDelete() {
	data := []byte("delete me")
	err := s.store.Archive(s.ctx, "delete-me.txt", bytes.NewReader(data), archive.ArchiveOptions{})
	s.Require().NoError(err)

	err = s.store.Delete(s.ctx, "delete-me.txt")
	s.Require().NoError(err)

	_, err = s.store.GetObject(s.ctx, "delete-me.txt")
	s.Error(err)
}

func (s *ArchiveStoreSuite) TestDeleteNotFound() {
	err := s.store.Delete(s.ctx, "nonexistent")
	s.Error(err)
}

func (s *ArchiveStoreSuite) TestGetObjectNotFound() {
	_, err := s.store.GetObject(s.ctx, "nonexistent")
	s.Error(err)
}

func (s *ArchiveStoreSuite) TestList() {
	// Archive some objects
	for i := 0; i < 5; i++ {
		key := "list/file" + string(rune('0'+i)) + ".txt"
		err := s.store.Archive(s.ctx, key, bytes.NewReader([]byte("data")), archive.ArchiveOptions{})
		s.Require().NoError(err)
	}

	// Also archive with different prefix
	err := s.store.Archive(s.ctx, "other/file.txt", bytes.NewReader([]byte("data")), archive.ArchiveOptions{})
	s.Require().NoError(err)

	// List all
	result, err := s.store.List(s.ctx, archive.ListOptions{})
	s.Require().NoError(err)
	s.Len(result.Objects, 6)

	// List with prefix
	result, err = s.store.List(s.ctx, archive.ListOptions{Prefix: "list/"})
	s.Require().NoError(err)
	s.Len(result.Objects, 5)
}

func (s *ArchiveStoreSuite) TestListWithPagination() {
	// Archive many objects
	for i := 0; i < 20; i++ {
		key := "page/file" + string(rune('a'+i)) + ".txt"
		err := s.store.Archive(s.ctx, key, bytes.NewReader([]byte("data")), archive.ArchiveOptions{})
		s.Require().NoError(err)
	}

	// First page
	result, err := s.store.List(s.ctx, archive.ListOptions{Prefix: "page/", Limit: 10})
	s.Require().NoError(err)
	s.Len(result.Objects, 10)
	s.True(result.IsTruncated)
	s.NotEmpty(result.NextContinuationToken)

	// Second page
	result, err = s.store.List(s.ctx, archive.ListOptions{
		Prefix:            "page/",
		Limit:             10,
		ContinuationToken: result.NextContinuationToken,
	})
	s.Require().NoError(err)
	s.Len(result.Objects, 10)
}

func (s *ArchiveStoreSuite) TestRestoreNonexistent() {
	_, err := s.store.Restore(s.ctx, "nonexistent", archive.RestoreOptions{})
	s.Error(err)
}

// TestArchiveStoreSuite runs the test suite.
func TestArchiveStoreSuite(t *testing.T) {
	suite.Run(t, new(ArchiveStoreSuite))
}

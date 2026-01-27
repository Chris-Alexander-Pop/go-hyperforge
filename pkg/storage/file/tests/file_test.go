package tests

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/storage/file"
	"github.com/chris-alexander-pop/system-design-library/pkg/storage/file/adapters/memory"
	"github.com/stretchr/testify/suite"
)

// FileStoreSuite provides a generic test suite for FileStore implementations.
type FileStoreSuite struct {
	suite.Suite
	store file.FileStore
	ctx   context.Context
}

// SetupTest runs before each test.
func (s *FileStoreSuite) SetupTest() {
	s.store = memory.New()
	s.ctx = context.Background()
}

func (s *FileStoreSuite) TestWriteAndRead() {
	content := []byte("hello, world!")
	err := s.store.Write(s.ctx, "/test/file.txt", bytes.NewReader(content))
	s.Require().NoError(err)

	rc, err := s.store.Read(s.ctx, "/test/file.txt")
	s.Require().NoError(err)
	defer rc.Close()

	data, err := io.ReadAll(rc)
	s.Require().NoError(err)
	s.Equal(content, data)
}

func (s *FileStoreSuite) TestReadNotFound() {
	_, err := s.store.Read(s.ctx, "/nonexistent.txt")
	s.Error(err)
}

func (s *FileStoreSuite) TestDelete() {
	content := []byte("to be deleted")
	err := s.store.Write(s.ctx, "/delete-me.txt", bytes.NewReader(content))
	s.Require().NoError(err)

	err = s.store.Delete(s.ctx, "/delete-me.txt")
	s.Require().NoError(err)

	_, err = s.store.Read(s.ctx, "/delete-me.txt")
	s.Error(err)
}

func (s *FileStoreSuite) TestDeleteNotFound() {
	err := s.store.Delete(s.ctx, "/nonexistent.txt")
	s.Error(err)
}

func (s *FileStoreSuite) TestStat() {
	content := []byte("file content here")
	err := s.store.Write(s.ctx, "/stat-test.txt", bytes.NewReader(content))
	s.Require().NoError(err)

	info, err := s.store.Stat(s.ctx, "/stat-test.txt")
	s.Require().NoError(err)
	s.Equal("/stat-test.txt", info.Path)
	s.Equal("stat-test.txt", info.Name)
	s.Equal(int64(len(content)), info.Size)
	s.False(info.IsDir)
}

func (s *FileStoreSuite) TestStatDirectory() {
	err := s.store.Mkdir(s.ctx, "/mydir")
	s.Require().NoError(err)

	info, err := s.store.Stat(s.ctx, "/mydir")
	s.Require().NoError(err)
	s.True(info.IsDir)
}

func (s *FileStoreSuite) TestMkdir() {
	err := s.store.Mkdir(s.ctx, "/a/b/c")
	s.Require().NoError(err)

	info, err := s.store.Stat(s.ctx, "/a/b/c")
	s.Require().NoError(err)
	s.True(info.IsDir)

	info, err = s.store.Stat(s.ctx, "/a/b")
	s.Require().NoError(err)
	s.True(info.IsDir)
}

func (s *FileStoreSuite) TestList() {
	// Create some files
	s.Require().NoError(s.store.Write(s.ctx, "/list/file1.txt", bytes.NewReader([]byte("1"))))
	s.Require().NoError(s.store.Write(s.ctx, "/list/file2.txt", bytes.NewReader([]byte("2"))))
	s.Require().NoError(s.store.Write(s.ctx, "/list/sub/file3.txt", bytes.NewReader([]byte("3"))))

	// List non-recursive
	files, err := s.store.List(s.ctx, "/list", file.ListOptions{Recursive: false})
	s.Require().NoError(err)
	s.Len(files, 3) // file1, file2, sub

	// List recursive
	files, err = s.store.List(s.ctx, "/list", file.ListOptions{Recursive: true})
	s.Require().NoError(err)
	s.Len(files, 4) // file1, file2, sub, file3
}

func (s *FileStoreSuite) TestListWithPagination() {
	// Create files
	for i := 0; i < 10; i++ {
		s.Require().NoError(s.store.Write(s.ctx, "/page/file"+string(rune('0'+i))+".txt", bytes.NewReader([]byte("x"))))
	}

	// List with limit
	files, err := s.store.List(s.ctx, "/page", file.ListOptions{Limit: 5})
	s.Require().NoError(err)
	s.Len(files, 5)

	// List with offset
	files, err = s.store.List(s.ctx, "/page", file.ListOptions{Offset: 8})
	s.Require().NoError(err)
	s.Len(files, 2)
}

func (s *FileStoreSuite) TestRename() {
	content := []byte("rename me")
	err := s.store.Write(s.ctx, "/old-name.txt", bytes.NewReader(content))
	s.Require().NoError(err)

	err = s.store.Rename(s.ctx, "/old-name.txt", "/new-name.txt")
	s.Require().NoError(err)

	// Old path should not exist
	_, err = s.store.Read(s.ctx, "/old-name.txt")
	s.Error(err)

	// New path should have the content
	rc, err := s.store.Read(s.ctx, "/new-name.txt")
	s.Require().NoError(err)
	defer rc.Close()

	data, err := io.ReadAll(rc)
	s.Require().NoError(err)
	s.Equal(content, data)
}

func (s *FileStoreSuite) TestCopy() {
	content := []byte("copy me")
	err := s.store.Write(s.ctx, "/original.txt", bytes.NewReader(content))
	s.Require().NoError(err)

	err = s.store.Copy(s.ctx, "/original.txt", "/copied.txt")
	s.Require().NoError(err)

	// Both should exist
	rc1, err := s.store.Read(s.ctx, "/original.txt")
	s.Require().NoError(err)
	defer rc1.Close()

	rc2, err := s.store.Read(s.ctx, "/copied.txt")
	s.Require().NoError(err)
	defer rc2.Close()

	data1, _ := io.ReadAll(rc1)
	data2, _ := io.ReadAll(rc2)
	s.Equal(data1, data2)
}

// TestFileStoreSuite runs the test suite.
func TestFileStoreSuite(t *testing.T) {
	suite.Run(t, new(FileStoreSuite))
}

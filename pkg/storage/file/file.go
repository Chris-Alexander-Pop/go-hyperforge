// Package file provides a unified interface for network file system storage.
//
// Supported backends:
//   - Memory: In-memory file system for testing
//   - NFS: Network File System (planned)
//   - EFS: AWS Elastic File System (planned)
//   - Azure Files: Azure file shares (planned)
//   - GCS FUSE: Google Cloud Storage FUSE (planned)
//
// Usage:
//
//	import "github.com/chris-alexander-pop/system-design-library/pkg/storage/file/adapters/memory"
//
//	store := memory.New()
//	err := store.Write(ctx, "/path/to/file.txt", reader)
//	reader, err := store.Read(ctx, "/path/to/file.txt")
package file

import (
	"context"
	"io"
	"time"
)

// Driver constants for file storage backends
const (
	DriverMemory     = "memory"
	DriverNFS        = "nfs"
	DriverEFS        = "efs"
	DriverAzureFiles = "azure-files"
	DriverGCSFuse    = "gcs-fuse"
)

// Config holds configuration for file storage.
type Config struct {
	// Driver specifies the file storage backend.
	Driver string `env:"FILE_DRIVER" env-default:"memory"`

	// MountPoint is the base path for file operations.
	MountPoint string `env:"FILE_MOUNT_POINT" env-default:"/mnt/data"`

	// NFS specific
	NFSServer string `env:"FILE_NFS_SERVER"`
	NFSPath   string `env:"FILE_NFS_PATH"`

	// EFS specific
	EFSID     string `env:"FILE_EFS_ID"`
	EFSRegion string `env:"FILE_EFS_REGION" env-default:"us-east-1"`

	// Azure Files specific
	AzureAccountName   string `env:"FILE_AZURE_ACCOUNT_NAME"`
	AzureAccountKey    string `env:"FILE_AZURE_ACCOUNT_KEY"`
	AzureShareName     string `env:"FILE_AZURE_SHARE_NAME"`

	// Common options
	MaxFileSize int64         `env:"FILE_MAX_SIZE" env-default:"1073741824"` // 1GB default
	Timeout     time.Duration `env:"FILE_TIMEOUT" env-default:"30s"`
}

// FileInfo contains metadata about a file or directory.
type FileInfo struct {
	// Path is the full path to the file.
	Path string

	// Name is the base name of the file.
	Name string

	// Size is the file size in bytes (0 for directories).
	Size int64

	// IsDir indicates if this is a directory.
	IsDir bool

	// ModTime is the last modification time.
	ModTime time.Time

	// Mode contains the file permission bits.
	Mode uint32

	// ContentType is the MIME type of the file (if known).
	ContentType string
}

// ListOptions configures file listing behavior.
type ListOptions struct {
	// Recursive lists files in subdirectories.
	Recursive bool

	// Limit is the maximum number of entries to return.
	Limit int

	// Offset is the starting offset for pagination.
	Offset int
}

// FileStore defines the interface for network file system operations.
type FileStore interface {
	// Read opens a file for reading.
	// Returns errors.NotFound if the file does not exist.
	Read(ctx context.Context, path string) (io.ReadCloser, error)

	// Write creates or overwrites a file with the given content.
	// Parent directories are created automatically if they don't exist.
	Write(ctx context.Context, path string, data io.Reader) error

	// Delete removes a file.
	// Returns errors.NotFound if the file does not exist.
	Delete(ctx context.Context, path string) error

	// List returns files and directories matching the prefix.
	List(ctx context.Context, prefix string, opts ListOptions) ([]FileInfo, error)

	// Stat returns metadata about a file or directory.
	// Returns errors.NotFound if the path does not exist.
	Stat(ctx context.Context, path string) (*FileInfo, error)

	// Mkdir creates a directory and all parent directories.
	Mkdir(ctx context.Context, path string) error

	// Rename moves or renames a file or directory.
	Rename(ctx context.Context, oldPath, newPath string) error

	// Copy copies a file to a new location.
	Copy(ctx context.Context, srcPath, dstPath string) error
}

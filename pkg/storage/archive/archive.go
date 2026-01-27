// Package archive provides a unified interface for cold/archive storage.
//
// Supported backends:
//   - Memory: In-memory archive store for testing
//   - Glacier: AWS S3 Glacier / Glacier Deep Archive (planned)
//   - Azure Archive: Azure Blob Archive tier (planned)
//   - GCS Archive: Google Cloud Storage Archive class (planned)
//
// Usage:
//
//	import "github.com/chris-alexander-pop/system-design-library/pkg/storage/archive/adapters/memory"
//
//	store := memory.New()
//	err := store.Archive(ctx, "backup/db-2024.tar.gz", reader)
//	job, err := store.Restore(ctx, "backup/db-2024.tar.gz")
package archive

import (
	"context"
	"io"
	"time"
)

// Driver constants for archive storage backends.
const (
	DriverMemory       = "memory"
	DriverGlacier      = "glacier"
	DriverGlacierDeep  = "glacier-deep"
	DriverAzureArchive = "azure-archive"
	DriverGCSArchive   = "gcs-archive"
)

// StorageClass represents the archive tier.
type StorageClass string

const (
	// StorageClassArchive is the standard archive tier (hours to restore).
	StorageClassArchive StorageClass = "archive"

	// StorageClassDeepArchive is the deep archive tier (12+ hours to restore).
	StorageClassDeepArchive StorageClass = "deep-archive"
)

// RestoreStatus represents the state of a restore operation.
type RestoreStatus string

const (
	RestoreStatusPending    RestoreStatus = "pending"
	RestoreStatusInProgress RestoreStatus = "in-progress"
	RestoreStatusCompleted  RestoreStatus = "completed"
	RestoreStatusFailed     RestoreStatus = "failed"
	RestoreStatusExpired    RestoreStatus = "expired"
)

// RestoreTier represents the speed of a restore operation.
type RestoreTier string

const (
	// RestoreTierExpedited is the fastest restore (1-5 minutes, more expensive).
	RestoreTierExpedited RestoreTier = "expedited"

	// RestoreTierStandard is the standard restore (3-5 hours).
	RestoreTierStandard RestoreTier = "standard"

	// RestoreTierBulk is the cheapest restore (5-12 hours).
	RestoreTierBulk RestoreTier = "bulk"
)

// Config holds configuration for archive storage.
type Config struct {
	// Driver specifies the archive storage backend.
	Driver string `env:"ARCHIVE_DRIVER" env-default:"memory"`

	// Bucket is the bucket/container name for archives.
	Bucket string `env:"ARCHIVE_BUCKET" env-default:"archive-bucket"`

	// Region is the cloud region.
	Region string `env:"ARCHIVE_REGION" env-default:"us-east-1"`

	// StorageClass is the default storage class for new archives.
	StorageClass StorageClass `env:"ARCHIVE_CLASS" env-default:"archive"`

	// AWS specific
	AWSAccessKeyID     string `env:"ARCHIVE_AWS_ACCESS_KEY"`
	AWSSecretAccessKey string `env:"ARCHIVE_AWS_SECRET_KEY"`

	// Azure specific
	AzureAccountName string `env:"ARCHIVE_AZURE_ACCOUNT"`
	AzureAccountKey  string `env:"ARCHIVE_AZURE_KEY"`

	// GCP specific
	GCPProjectID string `env:"ARCHIVE_GCP_PROJECT"`

	// Restore defaults
	DefaultRestoreTier RestoreTier   `env:"ARCHIVE_RESTORE_TIER" env-default:"standard"`
	RestoreTTL         time.Duration `env:"ARCHIVE_RESTORE_TTL" env-default:"168h"` // 7 days
}

// ArchiveObject represents an archived object.
type ArchiveObject struct {
	// Key is the unique identifier for the object.
	Key string

	// Size is the object size in bytes.
	Size int64

	// StorageClass is the current storage tier.
	StorageClass StorageClass

	// ArchivedAt is when the object was archived.
	ArchivedAt time.Time

	// RestoreStatus is the current restore state (if restoring).
	RestoreStatus RestoreStatus

	// RestoreExpiresAt is when the restored copy expires.
	RestoreExpiresAt time.Time

	// Checksum is the content checksum (SHA256).
	Checksum string

	// Metadata is user-defined key-value metadata.
	Metadata map[string]string
}

// RestoreJob represents a restore operation.
type RestoreJob struct {
	// ID is the unique identifier for the restore job.
	ID string

	// Key is the object being restored.
	Key string

	// Status is the current restore status.
	Status RestoreStatus

	// Tier is the restore speed tier.
	Tier RestoreTier

	// RequestedAt is when the restore was requested.
	RequestedAt time.Time

	// CompletedAt is when the restore completed (if completed).
	CompletedAt time.Time

	// ExpiresAt is when the restored copy expires.
	ExpiresAt time.Time

	// Error is the error message if the restore failed.
	Error string
}

// ArchiveOptions configures object archival.
type ArchiveOptions struct {
	// StorageClass overrides the default storage class.
	StorageClass StorageClass

	// Metadata is user-defined key-value metadata.
	Metadata map[string]string

	// ContentType is the MIME type of the content.
	ContentType string
}

// RestoreOptions configures object restoration.
type RestoreOptions struct {
	// Tier specifies the restore speed.
	Tier RestoreTier

	// TTL is how long to keep the restored copy available.
	TTL time.Duration
}

// ListOptions configures archive listing.
type ListOptions struct {
	// Prefix filters objects by key prefix.
	Prefix string

	// Limit is the maximum number of objects to return.
	Limit int

	// ContinuationToken is for pagination.
	ContinuationToken string
}

// ListResult contains the list result with pagination.
type ListResult struct {
	// Objects is the list of archived objects.
	Objects []*ArchiveObject

	// NextContinuationToken is the token for the next page.
	NextContinuationToken string

	// IsTruncated indicates if there are more results.
	IsTruncated bool
}

// ArchiveStore defines the interface for cold storage operations.
type ArchiveStore interface {
	// Archive stores data in cold storage.
	Archive(ctx context.Context, key string, data io.Reader, opts ArchiveOptions) error

	// Restore initiates a restore operation for an archived object.
	// Returns a RestoreJob that can be used to track progress.
	Restore(ctx context.Context, key string, opts RestoreOptions) (*RestoreJob, error)

	// GetRestoreStatus returns the current status of a restore operation.
	GetRestoreStatus(ctx context.Context, key string) (*RestoreJob, error)

	// Download retrieves a restored object.
	// Returns an error if the object has not been restored or restore has expired.
	Download(ctx context.Context, key string) (io.ReadCloser, error)

	// Delete removes an archived object.
	Delete(ctx context.Context, key string) error

	// GetObject returns metadata about an archived object.
	GetObject(ctx context.Context, key string) (*ArchiveObject, error)

	// List returns archived objects matching the options.
	List(ctx context.Context, opts ListOptions) (*ListResult, error)
}

// Package blob provides a unified interface for object/blob storage.
//
// Supported backends:
//   - Local: File-system based storage for development
//   - S3: AWS S3 compatible storage (works with MinIO, Wasabi, etc.)
//   - GCS: Google Cloud Storage
//   - Azure Blob: Azure Blob Storage
//
// Usage:
//
//	import "github.com/chris-alexander-pop/system-design-library/pkg/blob/adapters/s3"
//
//	store := s3.New(cfg)
//	err := store.Upload(ctx, "photos/cat.jpg", reader)
//	reader, err := store.Download(ctx, "photos/cat.jpg")
package blob

import (
	"context"
	"io"
)

// Config holds configuration for blob storage
type Config struct {
	Driver string `env:"BLOB_DRIVER" env-default:"local"` // local, s3
	Bucket string `env:"BLOB_BUCKET" env-default:"default-bucket"`

	// Local storage specific
	LocalDir string `env:"BLOB_LOCAL_DIR" env-default:"./uploads"`

	// S3 specific
	Region          string `env:"BLOB_S3_REGION" env-default:"us-east-1"`
	Endpoint        string `env:"BLOB_S3_ENDPOINT"` // optional, for minio/localstack
	AccessKeyID     string `env:"BLOB_S3_ACCESS_KEY"`
	SecretAccessKey string `env:"BLOB_S3_SECRET_KEY"`
}

// Store defines the interface for object storage
type Store interface {
	Upload(ctx context.Context, key string, data io.Reader) error
	Download(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
	// URL returns a public or internal URL for the key (optional, depends on implementation)
	URL(key string) string
}

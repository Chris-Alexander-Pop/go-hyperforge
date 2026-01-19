/*
Package blob provides a unified interface for object/blob storage.

Supported backends:
  - Local: File-system based storage
  - S3: AWS S3 compatible storage
  - GCS: Google Cloud Storage
  - Azure Blob: Azure Blob Storage
  - Memory: In-memory storage for testing

Usage:

	import "github.com/chris-alexander-pop/system-design-library/pkg/blob/adapters/s3"

	store := s3.New(cfg)
	err := store.Upload(ctx, "data.txt", reader)
*/
package blob

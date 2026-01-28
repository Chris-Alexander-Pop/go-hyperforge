/*
Package storage provides unified storage interfaces for file and object storage.

Subpackages:

  - archive: Archive file handling (tar, zip)
  - blob: Object/blob storage (S3, GCS, Azure Blob)
  - block: Block storage (EBS, Persistent Disk)
  - controller: Storage controller abstractions
  - file: File system operations

Usage:

	import "github.com/chris-alexander-pop/system-design-library/pkg/storage/blob"

	store, err := s3.New(cfg)
	err := store.Put(ctx, "bucket", "key", data)
*/
package storage

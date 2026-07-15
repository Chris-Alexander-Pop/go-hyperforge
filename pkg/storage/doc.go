/*
Package storage provides unified storage interfaces for object, file, block, and cold storage.

Subpackages:

  - blob: Object/blob storage (S3, GCS, Azure Blob, local, memory)
  - file: Network file system operations (memory adapter; cloud NFS/EFS planned)
  - block: Block/volume storage (memory adapter; EBS/Azure Disk/PD planned)
  - archive: Cold/archive storage such as Glacier-style tiers (not tar/zip packaging)
  - controller: Storage volume controller abstractions

Usage:

	import "github.com/chris-alexander-pop/go-hyperforge/pkg/storage/blob"
	import "github.com/chris-alexander-pop/go-hyperforge/pkg/storage/blob/adapters/s3"

	store, err := s3.New(ctx, cfg)
	err = store.Upload(ctx, "key", data)
*/
package storage

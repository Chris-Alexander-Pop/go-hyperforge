package storage

import (
	"github.com/chris-alexander-pop/go-hyperforge/pkg/config"
)

// Config is the root env-tagged configuration for storage capability drivers.
// Subpackages retain their own Config types; this aggregates common driver selection.
type Config struct {
	// BlobDriver selects blob backend: local, s3, gcs, azureblob, memory.
	BlobDriver string `env:"BLOB_DRIVER" env-default:"local"`

	// FileDriver selects file backend: memory, nfs, efs, etc.
	FileDriver string `env:"FILE_DRIVER" env-default:"memory"`

	// BlockDriver selects block backend: memory, ebs, etc.
	BlockDriver string `env:"BLOCK_DRIVER" env-default:"memory"`

	// ArchiveDriver selects cold-archive backend: memory, glacier, azure-archive, gcs-archive.
	ArchiveDriver string `env:"ARCHIVE_DRIVER" env-default:"memory"`

	// ControllerDriver selects volume controller: memory, lvm, local.
	ControllerDriver string `env:"STORAGE_DRIVER" env-default:"memory"`
}

// LoadConfig loads storage.Config via pkg/config (env / optional .env) and validates it.
func LoadConfig() (Config, error) {
	var cfg Config
	if err := config.Load(&cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

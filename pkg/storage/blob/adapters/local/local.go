package local

import (
	"context"
	"io"
	"os"
	"path/filepath"

	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"github.com/chris-alexander-pop/system-design-library/pkg/storage/blob"
)

// Store implements Store on the local filesystem
type Store struct {
	baseDir string
}

// New creates a new LocalStore
func New(cfg blob.Config) (*Store, error) {
	if cfg.LocalDir == "" {
		return nil, errors.New(errors.CodeInvalidArgument, "local dir is required", nil)
	}

	// Ensure directory exists
	if err := os.MkdirAll(cfg.LocalDir, 0755); err != nil {
		return nil, errors.Wrap(err, "failed to create blob directory")
	}

	return &Store{
		baseDir: cfg.LocalDir,
	}, nil
}

func (s *Store) Upload(ctx context.Context, key string, data io.Reader) error {
	fullPath := filepath.Join(s.baseDir, key)

	// Ensure parent dir exists
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return errors.Internal("failed to ensure blob dir", err)
	}

	f, err := os.Create(fullPath)
	if err != nil {
		return errors.Internal("failed to create blob file", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, data); err != nil {
		return errors.Internal("failed to write blob data", err)
	}

	return nil
}

func (s *Store) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	fullPath := filepath.Join(s.baseDir, key)

	f, err := os.Open(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.NotFound("blob not found", err)
		}
		return nil, errors.Internal("failed to open blob file", err)
	}

	return f, nil
}

func (s *Store) Delete(ctx context.Context, key string) error {
	fullPath := filepath.Join(s.baseDir, key)

	err := os.Remove(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return errors.NotFound("blob not found", err)
		}
		return errors.Internal("failed to delete blob file", err)
	}
	return nil
}

func (s *Store) URL(key string) string {
	// For local store, this might just be the file path or a mock URL
	return "file://" + filepath.Join(s.baseDir, key)
}

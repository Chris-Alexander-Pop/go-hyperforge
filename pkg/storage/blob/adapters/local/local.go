package local

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"github.com/chris-alexander-pop/system-design-library/pkg/storage/blob"
)

// Store implements Store on the local filesystem
type Store struct {
	baseDir string
}

// New creates a new LocalStore
func New(cfg blob.Config) (blob.Store, error) {
	if cfg.LocalDir == "" {
		return nil, errors.New(errors.CodeInvalidArgument, "local dir is required", nil)
	}

	// Ensure directory exists
	if err := os.MkdirAll(cfg.LocalDir, 0755); err != nil {
		return nil, errors.Wrap(err, "failed to create blob directory")
	}

	absDir, err := filepath.Abs(cfg.LocalDir)
	if err != nil {
		return nil, errors.Wrap(err, "failed to resolve absolute path for local dir")
	}

	return &Store{
		baseDir: filepath.Clean(absDir),
	}, nil
}

func (s *Store) validatePath(key string) (string, error) {
	// Join performs Clean implicitly
	fullPath := filepath.Join(s.baseDir, key)

	// Ensure the path is within the base directory
	// We append a separator to ensure we don't match partial directory names
	// e.g. /tmp/data vs /tmp/database
	prefix := s.baseDir
	if !strings.HasSuffix(prefix, string(os.PathSeparator)) {
		prefix += string(os.PathSeparator)
	}

	if !strings.HasPrefix(fullPath, prefix) {
		return "", errors.New(errors.CodeInvalidArgument, "invalid path: path traversal detected", nil)
	}

	return fullPath, nil
}

func (s *Store) Upload(ctx context.Context, key string, data io.Reader) error {
	fullPath, err := s.validatePath(key)
	if err != nil {
		return err
	}

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
	fullPath, err := s.validatePath(key)
	if err != nil {
		return nil, err
	}

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
	fullPath, err := s.validatePath(key)
	if err != nil {
		return err
	}

	err = os.Remove(fullPath)
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
	// Security: validate path to prevent traversal
	fullPath, err := s.validatePath(key)
	if err != nil {
		return ""
	}
	return "file://" + fullPath
}

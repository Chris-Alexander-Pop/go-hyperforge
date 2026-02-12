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

	return &Store{
		baseDir: filepath.Clean(cfg.LocalDir),
	}, nil
}

// resolvePath ensures the key resolves to a path within the base directory
func (s *Store) resolvePath(key string) (string, error) {
	// Join baseDir and key, then clean the path
	fullPath := filepath.Join(s.baseDir, key)

	// Special case for root directory
	if s.baseDir == string(os.PathSeparator) {
		return fullPath, nil
	}

	// Verify the path is within the base directory
	// We check if fullPath starts with baseDir + Separator to prevent "dir" matching "dir-suffix"
	// We also allow fullPath == baseDir if that's valid (usually not for file operations but harmless for directory checks)
	if !strings.HasPrefix(fullPath, s.baseDir+string(os.PathSeparator)) && fullPath != s.baseDir {
		return "", errors.New(errors.CodeInvalidArgument, "invalid key path: potential path traversal detected", nil)
	}

	return fullPath, nil
}

func (s *Store) Upload(ctx context.Context, key string, data io.Reader) error {
	fullPath, err := s.resolvePath(key)
	if err != nil {
		return err
	}

	// Ensure parent dir exists
	dir := filepath.Dir(fullPath)
	// Additional check: verify parent dir is also within baseDir (resolvePath handles fullPath so this should be safe)
	if !strings.HasPrefix(dir, s.baseDir) {
		return errors.New(errors.CodeInvalidArgument, "invalid parent directory", nil)
	}

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
	fullPath, err := s.resolvePath(key)
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
	fullPath, err := s.resolvePath(key)
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
	fullPath, err := s.resolvePath(key)
	if err != nil {
		// Return empty string or invalid URL indicator if path is unsafe
		return ""
	}
	return "file://" + fullPath
}

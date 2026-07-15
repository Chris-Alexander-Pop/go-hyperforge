// Package local provides a real local filesystem FileStore adapter.
//
// Paths are rooted at Config.MountPoint (or an explicit root passed to New).
// This adapter is also suitable for NFS/EFS-style mounts: point MountPoint at
// the already-mounted path; Hyperforge does not manage mount(8) itself.
package local

import (
	"context"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/storage/file"
)

// Store implements file.FileStore on a local (or NFS-mounted) directory tree.
type Store struct {
	root string
}

var _ file.FileStore = (*Store)(nil)

// New creates a Store rooted at rootDir (created if missing).
func New(rootDir string) (*Store, error) {
	if rootDir == "" {
		return nil, errors.InvalidArgument("file local root is required", nil)
	}
	if err := os.MkdirAll(rootDir, 0o755); err != nil {
		return nil, errors.Wrap(err, "failed to create file store root")
	}
	abs, err := filepath.Abs(rootDir)
	if err != nil {
		return nil, errors.Wrap(err, "failed to resolve file store root")
	}
	return &Store{root: filepath.Clean(abs)}, nil
}

// NewWithConfig uses cfg.MountPoint as the root directory.
func NewWithConfig(cfg file.Config) (*Store, error) {
	root := cfg.MountPoint
	if root == "" {
		root = "./filestore"
	}
	return New(root)
}

func (s *Store) resolve(p string) (string, error) {
	cleaned := path.Clean("/" + strings.TrimPrefix(p, "/"))
	rel := strings.TrimPrefix(cleaned, "/")
	full := filepath.Join(s.root, filepath.FromSlash(rel))
	full = filepath.Clean(full)

	prefix := s.root
	if !strings.HasSuffix(prefix, string(os.PathSeparator)) {
		prefix += string(os.PathSeparator)
	}
	if full != s.root && !strings.HasPrefix(full, prefix) {
		return "", errors.InvalidArgument("invalid path: path traversal detected", nil)
	}
	return full, nil
}

func (s *Store) Read(ctx context.Context, filePath string) (io.ReadCloser, error) {
	full, err := s.resolve(filePath)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(full)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.NotFound("file not found", err)
		}
		return nil, errors.Internal("failed to open file", err)
	}
	info, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return nil, errors.Internal("failed to stat file", err)
	}
	if info.IsDir() {
		_ = f.Close()
		return nil, errors.InvalidArgument("path is a directory", nil)
	}
	return f, nil
}

func (s *Store) Write(ctx context.Context, filePath string, data io.Reader) error {
	full, err := s.resolve(filePath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return errors.Internal("failed to create parent directories", err)
	}
	f, err := os.Create(full)
	if err != nil {
		return errors.Internal("failed to create file", err)
	}
	defer f.Close()
	if _, err := io.Copy(f, data); err != nil {
		return errors.Internal("failed to write file", err)
	}
	return nil
}

func (s *Store) Delete(ctx context.Context, filePath string) error {
	full, err := s.resolve(filePath)
	if err != nil {
		return err
	}
	if err := os.Remove(full); err != nil {
		if os.IsNotExist(err) {
			return errors.NotFound("file not found", err)
		}
		return errors.Internal("failed to delete file", err)
	}
	return nil
}

func (s *Store) List(ctx context.Context, prefix string, opts file.ListOptions) ([]file.FileInfo, error) {
	full, err := s.resolve(prefix)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(full)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, errors.Internal("failed to list path", err)
	}

	var results []file.FileInfo
	if !info.IsDir() {
		results = append(results, toInfo(prefix, info))
		return results, nil
	}

	walkFn := func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if p == full {
			return nil
		}
		if !opts.Recursive && filepath.Dir(p) != full {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		fi, err := d.Info()
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(s.root, p)
		if err != nil {
			return err
		}
		logical := "/" + filepath.ToSlash(rel)
		results = append(results, toInfo(logical, fi))
		if !opts.Recursive && d.IsDir() {
			return fs.SkipDir
		}
		return nil
	}

	if opts.Recursive {
		err = filepath.WalkDir(full, walkFn)
	} else {
		entries, readErr := os.ReadDir(full)
		if readErr != nil {
			return nil, errors.Internal("failed to read directory", readErr)
		}
		for _, e := range entries {
			fi, infoErr := e.Info()
			if infoErr != nil {
				return nil, errors.Internal("failed to stat entry", infoErr)
			}
			logical := path.Join(path.Clean("/"+strings.TrimPrefix(prefix, "/")), e.Name())
			if !strings.HasPrefix(logical, "/") {
				logical = "/" + logical
			}
			results = append(results, toInfo(logical, fi))
		}
	}
	if err != nil {
		return nil, errors.Internal("failed to walk directory", err)
	}

	if opts.Offset > 0 {
		if opts.Offset >= len(results) {
			return nil, nil
		}
		results = results[opts.Offset:]
	}
	if opts.Limit > 0 && len(results) > opts.Limit {
		results = results[:opts.Limit]
	}
	return results, nil
}

func (s *Store) Stat(ctx context.Context, filePath string) (*file.FileInfo, error) {
	full, err := s.resolve(filePath)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(full)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.NotFound("path not found", err)
		}
		return nil, errors.Internal("failed to stat path", err)
	}
	fi := toInfo(path.Clean("/"+strings.TrimPrefix(filePath, "/")), info)
	return &fi, nil
}

func (s *Store) Mkdir(ctx context.Context, filePath string) error {
	full, err := s.resolve(filePath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(full, 0o755); err != nil {
		return errors.Internal("failed to create directory", err)
	}
	return nil
}

func (s *Store) Rename(ctx context.Context, oldPath, newPath string) error {
	src, err := s.resolve(oldPath)
	if err != nil {
		return err
	}
	dst, err := s.resolve(newPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return errors.Internal("failed to create rename parent", err)
	}
	if err := os.Rename(src, dst); err != nil {
		if os.IsNotExist(err) {
			return errors.NotFound("path not found", err)
		}
		return errors.Internal("failed to rename", err)
	}
	return nil
}

func (s *Store) Copy(ctx context.Context, srcPath, dstPath string) error {
	src, err := s.resolve(srcPath)
	if err != nil {
		return err
	}
	dst, err := s.resolve(dstPath)
	if err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		if os.IsNotExist(err) {
			return errors.NotFound("file not found", err)
		}
		return errors.Internal("failed to open source", err)
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return errors.Internal("failed to create copy parent", err)
	}
	out, err := os.Create(dst)
	if err != nil {
		return errors.Internal("failed to create destination", err)
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return errors.Internal("failed to copy file", err)
	}
	return nil
}

func toInfo(logical string, info os.FileInfo) file.FileInfo {
	return file.FileInfo{
		Path:    logical,
		Name:    path.Base(logical),
		Size:    info.Size(),
		IsDir:   info.IsDir(),
		ModTime: info.ModTime().UTC(),
		Mode:    uint32(info.Mode().Perm()),
	}
}

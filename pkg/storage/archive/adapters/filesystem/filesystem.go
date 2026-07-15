// Package filesystem provides an archive.ArchiveStore backed by a cold directory
// on the local filesystem (objects under root/objects, restore metadata under root/meta).
package filesystem

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/concurrency"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/storage/archive"
	"github.com/google/uuid"
)

// meta is persisted restore/object metadata (sidecar JSON).
type meta struct {
	StorageClass archive.StorageClass `json:"storageClass"`
	ArchivedAt   time.Time            `json:"archivedAt"`
	Checksum     string               `json:"checksum"`
	Metadata     map[string]string    `json:"metadata"`
	ContentType  string               `json:"contentType"`
	RestoreJob   *archive.RestoreJob  `json:"restoreJob,omitempty"`
}

// Store implements archive.ArchiveStore on a filesystem cold directory.
type Store struct {
	root   string
	mu     *concurrency.SmartRWMutex
	config archive.Config
}

var _ archive.ArchiveStore = (*Store)(nil)

// New creates a filesystem cold-archive store at rootDir.
func New(rootDir string) (*Store, error) {
	return NewWithConfig(rootDir, archive.Config{
		StorageClass:       archive.StorageClassArchive,
		DefaultRestoreTier: archive.RestoreTierStandard,
		RestoreTTL:         168 * time.Hour,
	})
}

// NewWithConfig creates a store with explicit config.
func NewWithConfig(rootDir string, cfg archive.Config) (*Store, error) {
	if rootDir == "" {
		return nil, errors.InvalidArgument("archive filesystem root is required", nil)
	}
	for _, sub := range []string{"objects", "meta"} {
		if err := os.MkdirAll(filepath.Join(rootDir, sub), 0o755); err != nil {
			return nil, errors.Wrap(err, "failed to create archive dir")
		}
	}
	abs, err := filepath.Abs(rootDir)
	if err != nil {
		return nil, errors.Wrap(err, "failed to resolve archive root")
	}
	return &Store{
		root:   abs,
		mu:     concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "archive-filesystem"}),
		config: cfg,
	}, nil
}

func (s *Store) resolveObject(key string) (string, error) {
	cleaned := filepath.Clean("/" + strings.TrimPrefix(key, "/"))
	rel := strings.TrimPrefix(cleaned, "/")
	full := filepath.Join(s.root, "objects", filepath.FromSlash(rel))
	full = filepath.Clean(full)
	prefix := filepath.Join(s.root, "objects") + string(os.PathSeparator)
	if full != filepath.Join(s.root, "objects") && !strings.HasPrefix(full, prefix) {
		return "", errors.InvalidArgument("invalid key: path traversal detected", nil)
	}
	return full, nil
}

func (s *Store) metaPath(key string) (string, error) {
	cleaned := filepath.Clean("/" + strings.TrimPrefix(key, "/"))
	rel := strings.TrimPrefix(cleaned, "/")
	full := filepath.Join(s.root, "meta", filepath.FromSlash(rel)+".json")
	full = filepath.Clean(full)
	prefix := filepath.Join(s.root, "meta") + string(os.PathSeparator)
	if !strings.HasPrefix(full, prefix) {
		return "", errors.InvalidArgument("invalid key: path traversal detected", nil)
	}
	return full, nil
}

func (s *Store) writeMeta(key string, m *meta) error {
	path, err := s.metaPath(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return errors.Internal("create meta dir", err)
	}
	raw, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return errors.Internal("marshal meta", err)
	}
	return os.WriteFile(path, raw, 0o644)
}

func (s *Store) readMeta(key string) (*meta, error) {
	path, err := s.metaPath(key)
	if err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.NotFound("archived object not found", err)
		}
		return nil, errors.Internal("read meta", err)
	}
	var m meta
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, errors.Internal("unmarshal meta", err)
	}
	return &m, nil
}

func (s *Store) Archive(ctx context.Context, key string, data io.Reader, opts archive.ArchiveOptions) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	objPath, err := s.resolveObject(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(objPath), 0o755); err != nil {
		return errors.Internal("create object dir", err)
	}
	f, err := os.Create(objPath)
	if err != nil {
		return errors.Internal("create object", err)
	}
	hasher := sha256.New()
	n, err := io.Copy(io.MultiWriter(f, hasher), data)
	_ = f.Close()
	if err != nil {
		return errors.Internal("write object", err)
	}
	_ = n

	storageClass := opts.StorageClass
	if storageClass == "" {
		storageClass = s.config.StorageClass
	}
	m := &meta{
		StorageClass: storageClass,
		ArchivedAt:   time.Now().UTC(),
		Checksum:     hex.EncodeToString(hasher.Sum(nil)),
		Metadata:     opts.Metadata,
		ContentType:  opts.ContentType,
	}
	return s.writeMeta(key, m)
}

func (s *Store) Restore(ctx context.Context, key string, opts archive.RestoreOptions) (*archive.RestoreJob, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	m, err := s.readMeta(key)
	if err != nil {
		return nil, err
	}
	if m.RestoreJob != nil && m.RestoreJob.Status == archive.RestoreStatusInProgress {
		return nil, errors.Conflict("restore already in progress", nil)
	}
	tier := opts.Tier
	if tier == "" {
		tier = s.config.DefaultRestoreTier
	}
	ttl := opts.TTL
	if ttl == 0 {
		ttl = s.config.RestoreTTL
	}
	now := time.Now().UTC()
	job := &archive.RestoreJob{
		ID:          uuid.NewString(),
		Key:         key,
		Status:      archive.RestoreStatusCompleted, // local FS restore is instant
		Tier:        tier,
		RequestedAt: now,
		CompletedAt: now,
		ExpiresAt:   now.Add(ttl),
	}
	m.RestoreJob = job
	if err := s.writeMeta(key, m); err != nil {
		return nil, err
	}
	return job, nil
}

func (s *Store) GetRestoreStatus(ctx context.Context, key string) (*archive.RestoreJob, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	m, err := s.readMeta(key)
	if err != nil {
		return nil, err
	}
	if m.RestoreJob == nil {
		return nil, errors.NotFound("no restore job for object", nil)
	}
	if m.RestoreJob.Status == archive.RestoreStatusCompleted &&
		time.Now().After(m.RestoreJob.ExpiresAt) {
		m.RestoreJob.Status = archive.RestoreStatusExpired
	}
	return m.RestoreJob, nil
}

func (s *Store) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	m, err := s.readMeta(key)
	if err != nil {
		return nil, err
	}
	if m.RestoreJob == nil {
		return nil, errors.Conflict("object has not been restored", nil)
	}
	if m.RestoreJob.Status != archive.RestoreStatusCompleted {
		return nil, errors.Conflict("restore not completed", nil)
	}
	if time.Now().After(m.RestoreJob.ExpiresAt) {
		return nil, errors.Conflict("restored copy has expired", nil)
	}
	path, err := s.resolveObject(key)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.NotFound("archived object not found", err)
		}
		return nil, errors.Internal("open object", err)
	}
	return f, nil
}

func (s *Store) Delete(ctx context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	objPath, err := s.resolveObject(key)
	if err != nil {
		return err
	}
	metaPath, err := s.metaPath(key)
	if err != nil {
		return err
	}
	if _, err := os.Stat(metaPath); err != nil {
		if os.IsNotExist(err) {
			return errors.NotFound("archived object not found", err)
		}
		return errors.Internal("stat meta", err)
	}
	_ = os.Remove(objPath)
	if err := os.Remove(metaPath); err != nil {
		return errors.Internal("delete meta", err)
	}
	return nil
}

func (s *Store) GetObject(ctx context.Context, key string) (*archive.ArchiveObject, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	m, err := s.readMeta(key)
	if err != nil {
		return nil, err
	}
	path, err := s.resolveObject(key)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(path)
	size := int64(0)
	if err == nil {
		size = info.Size()
	}
	result := &archive.ArchiveObject{
		Key:          key,
		Size:         size,
		StorageClass: m.StorageClass,
		ArchivedAt:   m.ArchivedAt,
		Checksum:     m.Checksum,
		Metadata:     m.Metadata,
	}
	if m.RestoreJob != nil {
		result.RestoreStatus = m.RestoreJob.Status
		result.RestoreExpiresAt = m.RestoreJob.ExpiresAt
	}
	return result, nil
}

func (s *Store) List(ctx context.Context, opts archive.ListOptions) (*archive.ListResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	metaRoot := filepath.Join(s.root, "meta")
	var keys []string
	_ = filepath.WalkDir(metaRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if !strings.HasSuffix(path, ".json") {
			return nil
		}
		rel, err := filepath.Rel(metaRoot, path)
		if err != nil {
			return err
		}
		key := filepath.ToSlash(strings.TrimSuffix(rel, ".json"))
		if opts.Prefix == "" || strings.HasPrefix(key, opts.Prefix) {
			keys = append(keys, key)
		}
		return nil
	})
	sort.Strings(keys)

	startIdx := 0
	if opts.ContinuationToken != "" {
		for i, k := range keys {
			if k > opts.ContinuationToken {
				startIdx = i
				break
			}
		}
	}
	limit := opts.Limit
	if limit <= 0 {
		limit = 1000
	}

	result := &archive.ListResult{Objects: make([]*archive.ArchiveObject, 0)}
	for i := startIdx; i < len(keys) && len(result.Objects) < limit; i++ {
		key := keys[i]
		m, err := s.readMeta(key)
		if err != nil {
			continue
		}
		path, err := s.resolveObject(key)
		if err != nil {
			continue
		}
		size := int64(0)
		if info, err := os.Stat(path); err == nil {
			size = info.Size()
		}
		archiveObj := &archive.ArchiveObject{
			Key:          key,
			Size:         size,
			StorageClass: m.StorageClass,
			ArchivedAt:   m.ArchivedAt,
			Checksum:     m.Checksum,
			Metadata:     m.Metadata,
		}
		if m.RestoreJob != nil {
			archiveObj.RestoreStatus = m.RestoreJob.Status
			archiveObj.RestoreExpiresAt = m.RestoreJob.ExpiresAt
		}
		result.Objects = append(result.Objects, archiveObj)
	}
	if startIdx+len(result.Objects) < len(keys) {
		result.IsTruncated = true
		result.NextContinuationToken = result.Objects[len(result.Objects)-1].Key
	}
	return result, nil
}

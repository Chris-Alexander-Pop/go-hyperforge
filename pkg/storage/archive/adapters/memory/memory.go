package memory

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"github.com/chris-alexander-pop/system-design-library/pkg/storage/archive"
	"github.com/google/uuid"
)

// object represents an archived object in memory.
type object struct {
	data         []byte
	storageClass archive.StorageClass
	archivedAt   time.Time
	checksum     string
	metadata     map[string]string
	contentType  string

	// Restore state
	restoreJob *archive.RestoreJob
}

// Store implements an in-memory archive store for testing.
type Store struct {
	mu      sync.RWMutex
	objects map[string]*object
	config  archive.Config
}

// New creates a new in-memory archive store.
func New() *Store {
	return NewWithConfig(archive.Config{
		StorageClass:       archive.StorageClassArchive,
		DefaultRestoreTier: archive.RestoreTierStandard,
		RestoreTTL:         168 * time.Hour, // 7 days
	})
}

// NewWithConfig creates a new in-memory archive store with config.
func NewWithConfig(cfg archive.Config) *Store {
	return &Store{
		objects: make(map[string]*object),
		config:  cfg,
	}
}

func (s *Store) Archive(ctx context.Context, key string, data io.Reader, opts archive.ArchiveOptions) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	buf, err := io.ReadAll(data)
	if err != nil {
		return errors.Internal("failed to read data", err)
	}

	// Calculate checksum
	hash := sha256.Sum256(buf)
	checksum := hex.EncodeToString(hash[:])

	storageClass := opts.StorageClass
	if storageClass == "" {
		storageClass = s.config.StorageClass
	}

	s.objects[key] = &object{
		data:         buf,
		storageClass: storageClass,
		archivedAt:   time.Now(),
		checksum:     checksum,
		metadata:     opts.Metadata,
		contentType:  opts.ContentType,
	}

	return nil
}

func (s *Store) Restore(ctx context.Context, key string, opts archive.RestoreOptions) (*archive.RestoreJob, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	obj, ok := s.objects[key]
	if !ok {
		return nil, errors.NotFound("archived object not found", nil)
	}

	// Check if already restoring
	if obj.restoreJob != nil && obj.restoreJob.Status == archive.RestoreStatusInProgress {
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

	// For in-memory, restore is instant
	now := time.Now()
	job := &archive.RestoreJob{
		ID:          uuid.NewString(),
		Key:         key,
		Status:      archive.RestoreStatusCompleted,
		Tier:        tier,
		RequestedAt: now,
		CompletedAt: now,
		ExpiresAt:   now.Add(ttl),
	}

	obj.restoreJob = job
	return job, nil
}

func (s *Store) GetRestoreStatus(ctx context.Context, key string) (*archive.RestoreJob, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	obj, ok := s.objects[key]
	if !ok {
		return nil, errors.NotFound("archived object not found", nil)
	}

	if obj.restoreJob == nil {
		return nil, errors.NotFound("no restore job for object", nil)
	}

	// Check if expired
	if obj.restoreJob.Status == archive.RestoreStatusCompleted &&
		time.Now().After(obj.restoreJob.ExpiresAt) {
		obj.restoreJob.Status = archive.RestoreStatusExpired
	}

	return obj.restoreJob, nil
}

func (s *Store) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	obj, ok := s.objects[key]
	if !ok {
		return nil, errors.NotFound("archived object not found", nil)
	}

	if obj.restoreJob == nil {
		return nil, errors.Conflict("object has not been restored", nil)
	}

	if obj.restoreJob.Status != archive.RestoreStatusCompleted {
		return nil, errors.Conflict("restore not completed", nil)
	}

	if time.Now().After(obj.restoreJob.ExpiresAt) {
		return nil, errors.Conflict("restored copy has expired", nil)
	}

	return io.NopCloser(bytes.NewReader(obj.data)), nil
}

func (s *Store) Delete(ctx context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.objects[key]; !ok {
		return errors.NotFound("archived object not found", nil)
	}

	delete(s.objects, key)
	return nil
}

func (s *Store) GetObject(ctx context.Context, key string) (*archive.ArchiveObject, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	obj, ok := s.objects[key]
	if !ok {
		return nil, errors.NotFound("archived object not found", nil)
	}

	result := &archive.ArchiveObject{
		Key:          key,
		Size:         int64(len(obj.data)),
		StorageClass: obj.storageClass,
		ArchivedAt:   obj.archivedAt,
		Checksum:     obj.checksum,
		Metadata:     obj.metadata,
	}

	if obj.restoreJob != nil {
		result.RestoreStatus = obj.restoreJob.Status
		result.RestoreExpiresAt = obj.restoreJob.ExpiresAt
	}

	return result, nil
}

func (s *Store) List(ctx context.Context, opts archive.ListOptions) (*archive.ListResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := &archive.ListResult{
		Objects: make([]*archive.ArchiveObject, 0),
	}

	// Collect matching objects
	var keys []string
	for key := range s.objects {
		if opts.Prefix == "" || strings.HasPrefix(key, opts.Prefix) {
			keys = append(keys, key)
		}
	}

	// Sort for consistent ordering
	sort.Strings(keys)

	// Handle pagination via continuation token (simple offset-based for memory)
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

	for i := startIdx; i < len(keys) && len(result.Objects) < limit; i++ {
		key := keys[i]
		obj := s.objects[key]

		archiveObj := &archive.ArchiveObject{
			Key:          key,
			Size:         int64(len(obj.data)),
			StorageClass: obj.storageClass,
			ArchivedAt:   obj.archivedAt,
			Checksum:     obj.checksum,
			Metadata:     obj.metadata,
		}

		if obj.restoreJob != nil {
			archiveObj.RestoreStatus = obj.restoreJob.Status
			archiveObj.RestoreExpiresAt = obj.restoreJob.ExpiresAt
		}

		result.Objects = append(result.Objects, archiveObj)
	}

	// Check if truncated
	if startIdx+len(result.Objects) < len(keys) {
		result.IsTruncated = true
		result.NextContinuationToken = result.Objects[len(result.Objects)-1].Key
	}

	return result, nil
}

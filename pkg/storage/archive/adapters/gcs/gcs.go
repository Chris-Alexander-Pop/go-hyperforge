// Package gcs provides an archive.ArchiveStore backed by GCS ARCHIVE storage class.
package gcs

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"sort"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/storage"
	pkgerrors "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/storage/archive"
	"github.com/google/uuid"
	"google.golang.org/api/iterator"
)

const (
	storageClassArchive  = "ARCHIVE"
	storageClassColdline = "COLDLINE"
	storageClassStandard = "STANDARD"
)

// ObjectAPI is the GCS surface used by this adapter (for tests).
type ObjectAPI interface {
	Upload(ctx context.Context, key string, body io.Reader, storageClass string, metadata map[string]string, contentType string) (size int64, err error)
	Download(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
	Attrs(ctx context.Context, key string) (size int64, storageClass string, metadata map[string]string, archivedAt time.Time, err error)
	UpdateClass(ctx context.Context, key, storageClass string) error
	List(ctx context.Context, prefix string, limit int) ([]ObjectItem, error)
}

// ObjectItem is a listed object.
type ObjectItem struct {
	Key          string
	Size         int64
	StorageClass string
}

// Config configures the GCS archive adapter.
type Config struct {
	Bucket             string
	ProjectID          string
	StorageClass       archive.StorageClass
	DefaultRestoreTier archive.RestoreTier
	RestoreTTL         time.Duration
	InstantRestore     bool
}

// Store implements archive.ArchiveStore via GCS ARCHIVE class.
type Store struct {
	client ObjectAPI
	bucket string
	cfg    Config

	mu   sync.Mutex
	jobs map[string]*archive.RestoreJob
	meta map[string]objectMeta
}

type objectMeta struct {
	checksum     string
	metadata     map[string]string
	archivedAt   time.Time
	storageClass archive.StorageClass
	size         int64
}

var _ archive.ArchiveStore = (*Store)(nil)

// NewFromAPI wraps an ObjectAPI.
func NewFromAPI(api ObjectAPI, cfg Config) (*Store, error) {
	if api == nil {
		return nil, pkgerrors.InvalidArgument("gcs object api is required", nil)
	}
	if cfg.Bucket == "" {
		return nil, pkgerrors.InvalidArgument("archive bucket is required", nil)
	}
	if cfg.StorageClass == "" {
		cfg.StorageClass = archive.StorageClassArchive
	}
	if cfg.DefaultRestoreTier == "" {
		cfg.DefaultRestoreTier = archive.RestoreTierStandard
	}
	if cfg.RestoreTTL == 0 {
		cfg.RestoreTTL = 168 * time.Hour
	}
	return &Store{
		client: api,
		bucket: cfg.Bucket,
		cfg:    cfg,
		jobs:   make(map[string]*archive.RestoreJob),
		meta:   make(map[string]objectMeta),
	}, nil
}

// New builds a Store using Application Default Credentials.
func New(ctx context.Context, cfg Config) (*Store, error) {
	if cfg.Bucket == "" {
		return nil, pkgerrors.InvalidArgument("archive bucket is required", nil)
	}
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, pkgerrors.Internal("failed to create gcs client", err)
	}
	return NewFromAPI(&sdkObjectAPI{client: client, bucket: cfg.Bucket}, cfg)
}

// NewWithArchiveConfig maps archive.Config into the GCS adapter.
func NewWithArchiveConfig(ctx context.Context, ac archive.Config) (*Store, error) {
	return New(ctx, Config{
		Bucket:             ac.Bucket,
		ProjectID:          ac.GCPProjectID,
		StorageClass:       ac.StorageClass,
		DefaultRestoreTier: ac.DefaultRestoreTier,
		RestoreTTL:         ac.RestoreTTL,
	})
}

type sdkObjectAPI struct {
	client *storage.Client
	bucket string
}

func (s *sdkObjectAPI) Upload(ctx context.Context, key string, body io.Reader, storageClass string, metadata map[string]string, contentType string) (int64, error) {
	w := s.client.Bucket(s.bucket).Object(key).NewWriter(ctx)
	w.StorageClass = storageClass
	w.Metadata = metadata
	if contentType != "" {
		w.ContentType = contentType
	}
	n, err := io.Copy(w, body)
	if err != nil {
		_ = w.Close()
		return 0, err
	}
	if err := w.Close(); err != nil {
		return 0, err
	}
	return n, nil
}

func (s *sdkObjectAPI) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	return s.client.Bucket(s.bucket).Object(key).NewReader(ctx)
}

func (s *sdkObjectAPI) Delete(ctx context.Context, key string) error {
	return s.client.Bucket(s.bucket).Object(key).Delete(ctx)
}

func (s *sdkObjectAPI) Attrs(ctx context.Context, key string) (int64, string, map[string]string, time.Time, error) {
	attrs, err := s.client.Bucket(s.bucket).Object(key).Attrs(ctx)
	if err != nil {
		return 0, "", nil, time.Time{}, err
	}
	return attrs.Size, attrs.StorageClass, attrs.Metadata, attrs.Created, nil
}

func (s *sdkObjectAPI) UpdateClass(ctx context.Context, key, storageClass string) error {
	src := s.client.Bucket(s.bucket).Object(key)
	dst := s.client.Bucket(s.bucket).Object(key)
	copier := dst.CopierFrom(src)
	copier.StorageClass = storageClass
	_, err := copier.Run(ctx)
	return err
}

func (s *sdkObjectAPI) List(ctx context.Context, prefix string, limit int) ([]ObjectItem, error) {
	it := s.client.Bucket(s.bucket).Objects(ctx, &storage.Query{Prefix: prefix})
	var out []ObjectItem
	for {
		attrs, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		out = append(out, ObjectItem{Key: attrs.Name, Size: attrs.Size, StorageClass: attrs.StorageClass})
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (s *Store) gcsClass(c archive.StorageClass) string {
	if c == archive.StorageClassDeepArchive {
		return storageClassArchive
	}
	return storageClassArchive
}

func (s *Store) Archive(ctx context.Context, key string, data io.Reader, opts archive.ArchiveOptions) error {
	if key == "" {
		return pkgerrors.InvalidArgument("key is required", nil)
	}
	buf, err := io.ReadAll(data)
	if err != nil {
		return pkgerrors.Internal("failed to read archive body", err)
	}
	hash := sha256.Sum256(buf)
	checksum := hex.EncodeToString(hash[:])
	class := opts.StorageClass
	if class == "" {
		class = s.cfg.StorageClass
	}
	size, err := s.client.Upload(ctx, key, bytes.NewReader(buf), s.gcsClass(class), opts.Metadata, opts.ContentType)
	if err != nil {
		return pkgerrors.Unavailable("gcs archive upload failed", err)
	}
	s.mu.Lock()
	s.meta[key] = objectMeta{
		checksum:     checksum,
		metadata:     opts.Metadata,
		archivedAt:   time.Now().UTC(),
		storageClass: class,
		size:         size,
	}
	s.mu.Unlock()
	return nil
}

func (s *Store) Restore(ctx context.Context, key string, opts archive.RestoreOptions) (*archive.RestoreJob, error) {
	if _, err := s.GetObject(ctx, key); err != nil {
		return nil, err
	}
	s.mu.Lock()
	if job, ok := s.jobs[key]; ok && job.Status == archive.RestoreStatusInProgress {
		s.mu.Unlock()
		return nil, archive.ErrRestoreInProgress
	}
	s.mu.Unlock()

	tier := opts.Tier
	if tier == "" {
		tier = s.cfg.DefaultRestoreTier
	}
	ttl := opts.TTL
	if ttl == 0 {
		ttl = s.cfg.RestoreTTL
	}
	target := storageClassStandard
	if tier == archive.RestoreTierBulk {
		target = storageClassColdline
	}
	if err := s.client.UpdateClass(ctx, key, target); err != nil {
		return nil, pkgerrors.Unavailable("gcs restore (class update) failed", err)
	}

	now := time.Now().UTC()
	job := &archive.RestoreJob{
		ID:          uuid.NewString(),
		Key:         key,
		Status:      archive.RestoreStatusInProgress,
		Tier:        tier,
		RequestedAt: now,
		ExpiresAt:   now.Add(ttl),
	}
	if tier == archive.RestoreTierExpedited || s.cfg.InstantRestore {
		job.Status = archive.RestoreStatusCompleted
		job.CompletedAt = now
	}
	s.mu.Lock()
	s.jobs[key] = job
	s.mu.Unlock()
	return job, nil
}

// CompleteRestore marks an in-progress restore completed (test/operator hook).
func (s *Store) CompleteRestore(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	job, ok := s.jobs[key]
	if !ok {
		return pkgerrors.NotFound("no restore job for object", nil)
	}
	job.Status = archive.RestoreStatusCompleted
	job.CompletedAt = time.Now().UTC()
	return nil
}

func (s *Store) GetRestoreStatus(ctx context.Context, key string) (*archive.RestoreJob, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	job, ok := s.jobs[key]
	if !ok {
		return nil, pkgerrors.NotFound("no restore job for object", nil)
	}
	cp := *job
	if cp.Status == archive.RestoreStatusCompleted && time.Now().After(cp.ExpiresAt) {
		cp.Status = archive.RestoreStatusExpired
	}
	return &cp, nil
}

func (s *Store) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	job, err := s.GetRestoreStatus(ctx, key)
	if err != nil {
		return nil, archive.ErrObjectNotRestored
	}
	if job.Status != archive.RestoreStatusCompleted {
		return nil, archive.ErrObjectNotRestored
	}
	if time.Now().After(job.ExpiresAt) {
		return nil, archive.ErrRestoreExpired
	}
	rc, err := s.client.Download(ctx, key)
	if err != nil {
		return nil, mapErr("download", err)
	}
	return rc, nil
}

func (s *Store) Delete(ctx context.Context, key string) error {
	if err := s.client.Delete(ctx, key); err != nil {
		return mapErr("delete", err)
	}
	s.mu.Lock()
	delete(s.meta, key)
	delete(s.jobs, key)
	s.mu.Unlock()
	return nil
}

func (s *Store) GetObject(ctx context.Context, key string) (*archive.ArchiveObject, error) {
	s.mu.Lock()
	m, ok := s.meta[key]
	job := s.jobs[key]
	s.mu.Unlock()
	if ok {
		obj := &archive.ArchiveObject{
			Key:          key,
			Size:         m.size,
			StorageClass: m.storageClass,
			ArchivedAt:   m.archivedAt,
			Checksum:     m.checksum,
			Metadata:     m.metadata,
		}
		if job != nil {
			obj.RestoreStatus = job.Status
			obj.RestoreExpiresAt = job.ExpiresAt
		}
		return obj, nil
	}
	size, class, metadata, archivedAt, err := s.client.Attrs(ctx, key)
	if err != nil {
		return nil, mapErr("get", err)
	}
	sc := archive.StorageClassArchive
	if class != storageClassArchive {
		sc = archive.StorageClassArchive
	}
	_ = sc
	return &archive.ArchiveObject{
		Key:          key,
		Size:         size,
		StorageClass: archive.StorageClassArchive,
		ArchivedAt:   archivedAt,
		Metadata:     metadata,
	}, nil
}

func (s *Store) List(ctx context.Context, opts archive.ListOptions) (*archive.ListResult, error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = 1000
	}
	items, err := s.client.List(ctx, opts.Prefix, limit)
	if err != nil {
		return nil, pkgerrors.Unavailable("gcs list failed", err)
	}
	result := &archive.ListResult{Objects: make([]*archive.ArchiveObject, 0, len(items))}
	for _, item := range items {
		ao, err := s.GetObject(ctx, item.Key)
		if err != nil {
			ao = &archive.ArchiveObject{Key: item.Key, Size: item.Size, StorageClass: archive.StorageClassArchive}
		}
		result.Objects = append(result.Objects, ao)
	}
	return result, nil
}

func mapErr(op string, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, storage.ErrObjectNotExist) || errors.Is(err, storage.ErrBucketNotExist) {
		return archive.ErrObjectNotFound
	}
	return pkgerrors.Unavailable("failed to "+op+" gcs archive", err)
}

// MemoryObjectAPI is an in-process ObjectAPI for unit tests.
type MemoryObjectAPI struct {
	mu    sync.Mutex
	data  map[string][]byte
	meta  map[string]map[string]string
	class map[string]string
}

// NewMemoryObjectAPI creates a test double.
func NewMemoryObjectAPI() *MemoryObjectAPI {
	return &MemoryObjectAPI{
		data:  make(map[string][]byte),
		meta:  make(map[string]map[string]string),
		class: make(map[string]string),
	}
}

func (m *MemoryObjectAPI) Upload(_ context.Context, key string, body io.Reader, storageClass string, metadata map[string]string, _ string) (int64, error) {
	b, _ := io.ReadAll(body)
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = b
	m.meta[key] = metadata
	m.class[key] = storageClass
	return int64(len(b)), nil
}

func (m *MemoryObjectAPI) Download(_ context.Context, key string) (io.ReadCloser, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	b, ok := m.data[key]
	if !ok {
		return nil, storage.ErrObjectNotExist
	}
	return io.NopCloser(bytes.NewReader(b)), nil
}

func (m *MemoryObjectAPI) Delete(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.data[key]; !ok {
		return storage.ErrObjectNotExist
	}
	delete(m.data, key)
	delete(m.meta, key)
	delete(m.class, key)
	return nil
}

func (m *MemoryObjectAPI) Attrs(_ context.Context, key string) (int64, string, map[string]string, time.Time, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	b, ok := m.data[key]
	if !ok {
		return 0, "", nil, time.Time{}, storage.ErrObjectNotExist
	}
	return int64(len(b)), m.class[key], m.meta[key], time.Now().UTC(), nil
}

func (m *MemoryObjectAPI) UpdateClass(_ context.Context, key, storageClass string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.data[key]; !ok {
		return storage.ErrObjectNotExist
	}
	m.class[key] = storageClass
	return nil
}

func (m *MemoryObjectAPI) List(_ context.Context, prefix string, limit int) ([]ObjectItem, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	keys := make([]string, 0, len(m.data))
	for k := range m.data {
		if prefix == "" || strings.HasPrefix(k, prefix) {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	out := make([]ObjectItem, 0, len(keys))
	for _, k := range keys {
		out = append(out, ObjectItem{Key: k, Size: int64(len(m.data[k])), StorageClass: m.class[k]})
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}

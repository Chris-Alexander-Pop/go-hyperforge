// Package azure provides an archive.ArchiveStore backed by Azure Blob Archive tier.
package azure

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

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/bloberror"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/storage/archive"
	"github.com/google/uuid"
)

// BlobAPI is the Azure Blob surface used by this adapter (for tests).
type BlobAPI interface {
	UploadStream(ctx context.Context, containerName, blobName string, body io.Reader, o *azblob.UploadStreamOptions) error
	DownloadStream(ctx context.Context, containerName, blobName string) (io.ReadCloser, error)
	DeleteBlob(ctx context.Context, containerName, blobName string) error
	SetTier(ctx context.Context, containerName, blobName string, tier blob.AccessTier) error
	GetProperties(ctx context.Context, containerName, blobName string) (size int64, tier blob.AccessTier, metadata map[string]string, err error)
	ListBlobs(ctx context.Context, containerName, prefix string, limit int) ([]BlobItem, error)
}

// BlobItem is a listed blob.
type BlobItem struct {
	Name string
	Size int64
	Tier blob.AccessTier
}

// Config configures the Azure archive adapter.
type Config struct {
	AccountName        string
	AccountKey         string
	Container          string
	StorageClass       archive.StorageClass
	DefaultRestoreTier archive.RestoreTier
	RestoreTTL         time.Duration
	// InstantRestore completes restores immediately (tests / hot rehydrate simulation).
	InstantRestore bool
}

// Store implements archive.ArchiveStore via Azure Blob Archive access tier.
type Store struct {
	client    BlobAPI
	container string
	cfg       Config
	account   string

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

// NewFromAPI wraps a BlobAPI (SDK client wrapper or test double).
func NewFromAPI(api BlobAPI, cfg Config) (*Store, error) {
	if api == nil {
		return nil, errors.InvalidArgument("azure blob api is required", nil)
	}
	if cfg.Container == "" {
		return nil, errors.InvalidArgument("archive container is required", nil)
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
		client:    api,
		container: cfg.Container,
		cfg:       cfg,
		account:   cfg.AccountName,
		jobs:      make(map[string]*archive.RestoreJob),
		meta:      make(map[string]objectMeta),
	}, nil
}

// New builds a Store using DefaultAzureCredential against the account URL.
func New(ctx context.Context, cfg Config) (*Store, error) {
	_ = ctx
	if cfg.AccountName == "" {
		return nil, errors.InvalidArgument("azure account name is required", nil)
	}
	if cfg.Container == "" {
		return nil, errors.InvalidArgument("archive container is required", nil)
	}
	url := "https://" + cfg.AccountName + ".blob.core.windows.net/"
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, errors.Internal("failed to create azure credential", err)
	}
	client, err := azblob.NewClient(url, cred, nil)
	if err != nil {
		return nil, errors.Internal("failed to create azure blob client", err)
	}
	return NewFromAPI(&sdkBlobAPI{client: client}, cfg)
}

// NewWithArchiveConfig maps archive.Config into the Azure adapter.
func NewWithArchiveConfig(ctx context.Context, ac archive.Config) (*Store, error) {
	return New(ctx, Config{
		AccountName:        ac.AzureAccountName,
		AccountKey:         ac.AzureAccountKey,
		Container:          ac.Bucket,
		StorageClass:       ac.StorageClass,
		DefaultRestoreTier: ac.DefaultRestoreTier,
		RestoreTTL:         ac.RestoreTTL,
	})
}

type sdkBlobAPI struct {
	client *azblob.Client
}

func (s *sdkBlobAPI) UploadStream(ctx context.Context, containerName, blobName string, body io.Reader, o *azblob.UploadStreamOptions) error {
	_, err := s.client.UploadStream(ctx, containerName, blobName, body, o)
	return err
}

func (s *sdkBlobAPI) DownloadStream(ctx context.Context, containerName, blobName string) (io.ReadCloser, error) {
	resp, err := s.client.DownloadStream(ctx, containerName, blobName, nil)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

func (s *sdkBlobAPI) DeleteBlob(ctx context.Context, containerName, blobName string) error {
	_, err := s.client.DeleteBlob(ctx, containerName, blobName, nil)
	return err
}

func (s *sdkBlobAPI) SetTier(ctx context.Context, containerName, blobName string, tier blob.AccessTier) error {
	_, err := s.client.ServiceClient().NewContainerClient(containerName).NewBlobClient(blobName).SetTier(ctx, tier, nil)
	return err
}

func (s *sdkBlobAPI) GetProperties(ctx context.Context, containerName, blobName string) (int64, blob.AccessTier, map[string]string, error) {
	props, err := s.client.ServiceClient().NewContainerClient(containerName).NewBlobClient(blobName).GetProperties(ctx, nil)
	if err != nil {
		return 0, "", nil, err
	}
	size := int64(0)
	if props.ContentLength != nil {
		size = *props.ContentLength
	}
	tier := blob.AccessTierHot
	if props.AccessTier != nil {
		tier = blob.AccessTier(*props.AccessTier)
	}
	meta := map[string]string{}
	for k, v := range props.Metadata {
		if v != nil {
			meta[k] = *v
		}
	}
	return size, tier, meta, nil
}

func (s *sdkBlobAPI) ListBlobs(ctx context.Context, containerName, prefix string, limit int) ([]BlobItem, error) {
	pager := s.client.NewListBlobsFlatPager(containerName, &azblob.ListBlobsFlatOptions{
		Prefix: to.Ptr(prefix),
	})
	var out []BlobItem
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, item := range page.Segment.BlobItems {
			if item == nil || item.Name == nil {
				continue
			}
			bi := BlobItem{Name: *item.Name}
			if item.Properties != nil && item.Properties.ContentLength != nil {
				bi.Size = *item.Properties.ContentLength
			}
			if item.Properties != nil && item.Properties.AccessTier != nil {
				bi.Tier = blob.AccessTier(*item.Properties.AccessTier)
			}
			out = append(out, bi)
			if limit > 0 && len(out) >= limit {
				return out, nil
			}
		}
	}
	return out, nil
}

func (s *Store) Archive(ctx context.Context, key string, data io.Reader, opts archive.ArchiveOptions) error {
	if key == "" {
		return errors.InvalidArgument("key is required", nil)
	}
	buf, err := io.ReadAll(data)
	if err != nil {
		return errors.Internal("failed to read archive body", err)
	}
	hash := sha256.Sum256(buf)
	checksum := hex.EncodeToString(hash[:])
	meta := map[string]*string{}
	for k, v := range opts.Metadata {
		meta[k] = to.Ptr(v)
	}
	uploadOpts := &azblob.UploadStreamOptions{Metadata: meta}
	if err := s.client.UploadStream(ctx, s.container, key, bytes.NewReader(buf), uploadOpts); err != nil {
		return errors.Unavailable("azure archive upload failed", err)
	}
	if err := s.client.SetTier(ctx, s.container, key, blob.AccessTierArchive); err != nil {
		return errors.Unavailable("azure set archive tier failed", err)
	}
	class := opts.StorageClass
	if class == "" {
		class = s.cfg.StorageClass
	}
	s.mu.Lock()
	s.meta[key] = objectMeta{
		checksum:     checksum,
		metadata:     opts.Metadata,
		archivedAt:   time.Now().UTC(),
		storageClass: class,
		size:         int64(len(buf)),
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
	// Rehydrate to Hot (standard) or Cool (bulk-ish).
	access := blob.AccessTierHot
	if tier == archive.RestoreTierBulk {
		access = blob.AccessTierCool
	}
	if err := s.client.SetTier(ctx, s.container, key, access); err != nil {
		return nil, errors.Unavailable("azure rehydrate failed", err)
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
		return errors.NotFound("no restore job for object", nil)
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
		return nil, errors.NotFound("no restore job for object", nil)
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
	rc, err := s.client.DownloadStream(ctx, s.container, key)
	if err != nil {
		return nil, mapErr("download", err)
	}
	return rc, nil
}

func (s *Store) Delete(ctx context.Context, key string) error {
	if err := s.client.DeleteBlob(ctx, s.container, key); err != nil {
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
	size, tier, metadata, err := s.client.GetProperties(ctx, s.container, key)
	if err != nil {
		return nil, mapErr("get", err)
	}
	class := archive.StorageClassArchive
	if tier != blob.AccessTierArchive {
		class = archive.StorageClassArchive
	}
	_ = class
	return &archive.ArchiveObject{
		Key:          key,
		Size:         size,
		StorageClass: archive.StorageClassArchive,
		Metadata:     metadata,
	}, nil
}

func (s *Store) List(ctx context.Context, opts archive.ListOptions) (*archive.ListResult, error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = 1000
	}
	items, err := s.client.ListBlobs(ctx, s.container, opts.Prefix, limit)
	if err != nil {
		return nil, errors.Unavailable("azure list failed", err)
	}
	result := &archive.ListResult{Objects: make([]*archive.ArchiveObject, 0, len(items))}
	for _, item := range items {
		ao, err := s.GetObject(ctx, item.Name)
		if err != nil {
			ao = &archive.ArchiveObject{Key: item.Name, Size: item.Size, StorageClass: archive.StorageClassArchive}
		}
		result.Objects = append(result.Objects, ao)
	}
	return result, nil
}

func mapErr(op string, err error) error {
	if err == nil {
		return nil
	}
	if bloberror.HasCode(err, bloberror.BlobNotFound, bloberror.ContainerNotFound) {
		return archive.ErrObjectNotFound
	}
	var respErr *azcore.ResponseError
	if errors.As(err, &respErr) && (respErr.ErrorCode == string(bloberror.BlobNotFound) || respErr.ErrorCode == string(bloberror.ContainerNotFound)) {
		return archive.ErrObjectNotFound
	}
	return errors.Unavailable("failed to "+op+" azure archive", err)
}

// MemoryBlobAPI is an in-process BlobAPI for unit tests.
type MemoryBlobAPI struct {
	mu   sync.Mutex
	data map[string][]byte
	meta map[string]map[string]string
	tier map[string]blob.AccessTier
}

// NewMemoryBlobAPI creates a test double.
func NewMemoryBlobAPI() *MemoryBlobAPI {
	return &MemoryBlobAPI{
		data: make(map[string][]byte),
		meta: make(map[string]map[string]string),
		tier: make(map[string]blob.AccessTier),
	}
}

func (m *MemoryBlobAPI) UploadStream(_ context.Context, _, blobName string, body io.Reader, o *azblob.UploadStreamOptions) error {
	b, _ := io.ReadAll(body)
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[blobName] = b
	if o != nil && o.Metadata != nil {
		md := map[string]string{}
		for k, v := range o.Metadata {
			if v != nil {
				md[k] = *v
			}
		}
		m.meta[blobName] = md
	}
	m.tier[blobName] = blob.AccessTierHot
	return nil
}

func (m *MemoryBlobAPI) DownloadStream(_ context.Context, _, blobName string) (io.ReadCloser, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	b, ok := m.data[blobName]
	if !ok {
		return nil, &azcore.ResponseError{ErrorCode: string(bloberror.BlobNotFound)}
	}
	return io.NopCloser(bytes.NewReader(b)), nil
}

func (m *MemoryBlobAPI) DeleteBlob(_ context.Context, _, blobName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, blobName)
	delete(m.meta, blobName)
	delete(m.tier, blobName)
	return nil
}

func (m *MemoryBlobAPI) SetTier(_ context.Context, _, blobName string, tier blob.AccessTier) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.data[blobName]; !ok {
		return &azcore.ResponseError{ErrorCode: string(bloberror.BlobNotFound)}
	}
	m.tier[blobName] = tier
	return nil
}

func (m *MemoryBlobAPI) GetProperties(_ context.Context, _, blobName string) (int64, blob.AccessTier, map[string]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	b, ok := m.data[blobName]
	if !ok {
		return 0, "", nil, &azcore.ResponseError{ErrorCode: string(bloberror.BlobNotFound)}
	}
	return int64(len(b)), m.tier[blobName], m.meta[blobName], nil
}

func (m *MemoryBlobAPI) ListBlobs(_ context.Context, _, prefix string, limit int) ([]BlobItem, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	keys := make([]string, 0, len(m.data))
	for k := range m.data {
		if prefix == "" || strings.HasPrefix(k, prefix) {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	out := make([]BlobItem, 0, len(keys))
	for _, k := range keys {
		out = append(out, BlobItem{Name: k, Size: int64(len(m.data[k])), Tier: m.tier[k]})
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}

package glacier

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/storage/archive"
	"github.com/google/uuid"
)

// ObjectAPI is the S3 surface used by this adapter.
type ObjectAPI interface {
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
	HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error)
	ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error)
	RestoreObject(ctx context.Context, params *s3.RestoreObjectInput, optFns ...func(*s3.Options)) (*s3.RestoreObjectOutput, error)
}

// Config configures the Glacier thin adapter.
type Config struct {
	Bucket             string `env:"ARCHIVE_BUCKET" env-default:"archive-bucket"`
	Region             string `env:"ARCHIVE_REGION" env-default:"us-east-1"`
	StorageClass       archive.StorageClass
	DefaultRestoreTier archive.RestoreTier
	RestoreTTL         time.Duration
	AWSAccessKeyID     string
	AWSSecretAccessKey string
	Endpoint           string
}

// Store implements archive.ArchiveStore via S3 Glacier storage classes.
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

// NewFromAPI wraps an ObjectAPI (SDK client or test double).
func NewFromAPI(api ObjectAPI, cfg Config) (*Store, error) {
	if api == nil {
		return nil, errors.InvalidArgument("glacier object api is required", nil)
	}
	if cfg.Bucket == "" {
		return nil, errors.InvalidArgument("archive bucket is required", nil)
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

// New builds a Store from AWS SDK config.
func New(ctx context.Context, cfg Config) (*Store, error) {
	if cfg.Region == "" {
		return nil, errors.InvalidArgument("aws region is required", nil)
	}
	opts := []func(*config.LoadOptions) error{
		config.WithRegion(cfg.Region),
	}
	if cfg.AWSAccessKeyID != "" && cfg.AWSSecretAccessKey != "" {
		opts = append(opts, config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AWSAccessKeyID, cfg.AWSSecretAccessKey, "",
		)))
	}
	awsCfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, errors.Unavailable("failed to load aws config", err)
	}
	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if cfg.Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
			o.UsePathStyle = true
		}
	})
	return NewFromAPI(client, cfg)
}

func (s *Store) s3Class(c archive.StorageClass) types.StorageClass {
	if c == archive.StorageClassDeepArchive {
		return types.StorageClassDeepArchive
	}
	return types.StorageClassGlacier
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
	class := opts.StorageClass
	if class == "" {
		class = s.cfg.StorageClass
	}
	input := &s3.PutObjectInput{
		Bucket:       aws.String(s.bucket),
		Key:          aws.String(key),
		Body:         bytes.NewReader(buf),
		StorageClass: s.s3Class(class),
		Metadata:     opts.Metadata,
	}
	if opts.ContentType != "" {
		input.ContentType = aws.String(opts.ContentType)
	}
	if _, err := s.client.PutObject(ctx, input); err != nil {
		return errors.Unavailable("glacier put failed", err)
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
	days := int32(ttl.Hours() / 24)
	if days < 1 {
		days = 1
	}
	glacierTier := types.TierStandard
	switch tier {
	case archive.RestoreTierExpedited:
		glacierTier = types.TierExpedited
	case archive.RestoreTierBulk:
		glacierTier = types.TierBulk
	}

	_, err := s.client.RestoreObject(ctx, &s3.RestoreObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
		RestoreRequest: &types.RestoreRequest{
			Days: aws.Int32(days),
			GlacierJobParameters: &types.GlacierJobParameters{
				Tier: glacierTier,
			},
		},
	})
	if err != nil {
		return nil, errors.Unavailable("glacier restore failed", err)
	}

	now := time.Now().UTC()
	job := &archive.RestoreJob{
		ID:          uuid.NewString(),
		Key:         key,
		Status:      archive.RestoreStatusCompleted, // thin adapter: treat success as completed
		Tier:        tier,
		RequestedAt: now,
		CompletedAt: now,
		ExpiresAt:   now.Add(ttl),
	}
	s.mu.Lock()
	s.jobs[key] = job
	s.mu.Unlock()
	return job, nil
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
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, errors.Unavailable("glacier get failed", err)
	}
	return out.Body, nil
}

func (s *Store) Delete(ctx context.Context, key string) error {
	if _, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}); err != nil {
		return errors.Unavailable("glacier delete failed", err)
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
	head, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, archive.ErrObjectNotFound
	}
	size := int64(0)
	if head.ContentLength != nil {
		size = *head.ContentLength
	}
	class := archive.StorageClassArchive
	if head.StorageClass == types.StorageClassDeepArchive {
		class = archive.StorageClassDeepArchive
	}
	return &archive.ArchiveObject{
		Key:          key,
		Size:         size,
		StorageClass: class,
		Metadata:     head.Metadata,
	}, nil
}

func (s *Store) List(ctx context.Context, opts archive.ListOptions) (*archive.ListResult, error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = 1000
	}
	out, err := s.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket:            aws.String(s.bucket),
		Prefix:            aws.String(opts.Prefix),
		MaxKeys:           aws.Int32(int32(limit)),
		ContinuationToken: awsStringPtr(opts.ContinuationToken),
	})
	if err != nil {
		return nil, errors.Unavailable("glacier list failed", err)
	}
	result := &archive.ListResult{Objects: make([]*archive.ArchiveObject, 0, len(out.Contents))}
	for _, obj := range out.Contents {
		key := aws.ToString(obj.Key)
		ao, err := s.GetObject(ctx, key)
		if err != nil {
			ao = &archive.ArchiveObject{Key: key, Size: aws.ToInt64(obj.Size)}
		}
		result.Objects = append(result.Objects, ao)
	}
	if out.IsTruncated != nil && *out.IsTruncated {
		result.IsTruncated = true
		result.NextContinuationToken = aws.ToString(out.NextContinuationToken)
	}
	return result, nil
}

func awsStringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return aws.String(s)
}

// MemoryObjectAPI is a test double implementing ObjectAPI in-process.
type MemoryObjectAPI struct {
	mu    sync.Mutex
	data  map[string][]byte
	meta  map[string]map[string]string
	class map[string]types.StorageClass
}

// NewMemoryObjectAPI creates an in-memory ObjectAPI for tests.
func NewMemoryObjectAPI() *MemoryObjectAPI {
	return &MemoryObjectAPI{
		data:  make(map[string][]byte),
		meta:  make(map[string]map[string]string),
		class: make(map[string]types.StorageClass),
	}
}

func (m *MemoryObjectAPI) PutObject(ctx context.Context, params *s3.PutObjectInput, _ ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	_ = ctx
	body, _ := io.ReadAll(params.Body)
	key := aws.ToString(params.Key)
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = body
	m.meta[key] = params.Metadata
	m.class[key] = params.StorageClass
	return &s3.PutObjectOutput{}, nil
}

func (m *MemoryObjectAPI) GetObject(ctx context.Context, params *s3.GetObjectInput, _ ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	_ = ctx
	m.mu.Lock()
	defer m.mu.Unlock()
	b, ok := m.data[aws.ToString(params.Key)]
	if !ok {
		return nil, &types.NoSuchKey{}
	}
	return &s3.GetObjectOutput{Body: io.NopCloser(bytes.NewReader(b))}, nil
}

func (m *MemoryObjectAPI) DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, _ ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	_ = ctx
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, aws.ToString(params.Key))
	return &s3.DeleteObjectOutput{}, nil
}

func (m *MemoryObjectAPI) HeadObject(ctx context.Context, params *s3.HeadObjectInput, _ ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	_ = ctx
	m.mu.Lock()
	defer m.mu.Unlock()
	b, ok := m.data[aws.ToString(params.Key)]
	if !ok {
		return nil, &types.NotFound{}
	}
	n := int64(len(b))
	return &s3.HeadObjectOutput{
		ContentLength: &n,
		StorageClass:  m.class[aws.ToString(params.Key)],
		Metadata:      m.meta[aws.ToString(params.Key)],
	}, nil
}

func (m *MemoryObjectAPI) ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, _ ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
	_ = ctx
	m.mu.Lock()
	defer m.mu.Unlock()
	prefix := aws.ToString(params.Prefix)
	var contents []types.Object
	for k, v := range m.data {
		if prefix != "" && !strings.HasPrefix(k, prefix) {
			continue
		}
		n := int64(len(v))
		contents = append(contents, types.Object{Key: aws.String(k), Size: &n})
	}
	return &s3.ListObjectsV2Output{Contents: contents}, nil
}

func (m *MemoryObjectAPI) RestoreObject(ctx context.Context, params *s3.RestoreObjectInput, _ ...func(*s3.Options)) (*s3.RestoreObjectOutput, error) {
	_ = ctx
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.data[aws.ToString(params.Key)]; !ok {
		return nil, &types.NoSuchKey{}
	}
	return &s3.RestoreObjectOutput{}, nil
}

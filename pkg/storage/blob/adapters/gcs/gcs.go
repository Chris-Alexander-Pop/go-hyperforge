package gcs

import (
	"context"
	"errors"
	"fmt"
	"io"

	"cloud.google.com/go/storage"
	pkgerrors "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/storage/blob"
)

// Ensure Store implements blob.Store.
var _ blob.Store = (*Store)(nil)

// Store implements blob.Store using Google Cloud Storage.
type Store struct {
	client *storage.Client
	bucket string
}

// New creates a GCS-backed blob store using Application Default Credentials.
func New(ctx context.Context, cfg blob.Config) (blob.Store, error) {
	if cfg.Bucket == "" {
		return nil, blob.ErrInvalidConfig
	}

	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, pkgerrors.Internal("failed to create gcs client", err)
	}

	return &Store{
		client: client,
		bucket: cfg.Bucket,
	}, nil
}

func mapError(op string, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, storage.ErrObjectNotExist) || errors.Is(err, storage.ErrBucketNotExist) {
		return pkgerrors.NotFound("blob not found", err)
	}
	return pkgerrors.Internal("failed to "+op+" from gcs", err)
}

func (s *Store) Upload(ctx context.Context, key string, data io.Reader) error {
	w := s.client.Bucket(s.bucket).Object(key).NewWriter(ctx)
	if _, err := io.Copy(w, data); err != nil {
		_ = w.Close()
		return mapError("upload", err)
	}
	if err := w.Close(); err != nil {
		return mapError("upload", err)
	}
	return nil
}

func (s *Store) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	r, err := s.client.Bucket(s.bucket).Object(key).NewReader(ctx)
	if err != nil {
		return nil, mapError("download", err)
	}
	return r, nil
}

func (s *Store) Delete(ctx context.Context, key string) error {
	err := s.client.Bucket(s.bucket).Object(key).Delete(ctx)
	if err != nil {
		return mapError("delete", err)
	}
	return nil
}

func (s *Store) URL(key string) string {
	return fmt.Sprintf("https://storage.googleapis.com/%s/%s", s.bucket, key)
}

// Close releases the underlying GCS client.
func (s *Store) Close() error {
	if s.client == nil {
		return nil
	}
	return s.client.Close()
}

package azureblob

import (
	"context"
	"fmt"
	"io"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/bloberror"
	pkgerrors "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/storage/blob"
)

// Ensure Store implements blob.Store.
var _ blob.Store = (*Store)(nil)

// Store implements blob.Store using Azure Blob Storage.
type Store struct {
	client    *azblob.Client
	container string
	account   string
}

// New creates an Azure Blob-backed store.
// Uses Config.AzureAccountName (or falls back to AccessKeyID) and Config.Bucket as the container.
func New(cfg blob.Config) (blob.Store, error) {
	account := cfg.AzureAccountName
	if account == "" {
		account = cfg.AccessKeyID
	}
	if account == "" {
		return nil, blob.ErrInvalidConfig
	}
	if cfg.Bucket == "" {
		return nil, blob.ErrInvalidConfig
	}

	url := fmt.Sprintf("https://%s.blob.core.windows.net/", account)

	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, pkgerrors.Internal("failed to create azure credential", err)
	}

	client, err := azblob.NewClient(url, cred, nil)
	if err != nil {
		return nil, pkgerrors.Internal("failed to create azure blob client", err)
	}

	return &Store{
		client:    client,
		container: cfg.Bucket,
		account:   account,
	}, nil
}

func mapError(op string, err error) error {
	if err == nil {
		return nil
	}
	if bloberror.HasCode(err, bloberror.BlobNotFound, bloberror.ContainerNotFound) {
		return pkgerrors.NotFound("blob not found", err)
	}
	return pkgerrors.Internal("failed to "+op+" from azure blob", err)
}

func (s *Store) Upload(ctx context.Context, key string, data io.Reader) error {
	_, err := s.client.UploadStream(ctx, s.container, key, data, nil)
	if err != nil {
		return mapError("upload", err)
	}
	return nil
}

func (s *Store) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	resp, err := s.client.DownloadStream(ctx, s.container, key, nil)
	if err != nil {
		return nil, mapError("download", err)
	}
	return resp.Body, nil
}

func (s *Store) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteBlob(ctx, s.container, key, nil)
	if err != nil {
		return mapError("delete", err)
	}
	return nil
}

func (s *Store) URL(key string) string {
	return fmt.Sprintf("https://%s.blob.core.windows.net/%s/%s", s.account, s.container, key)
}

// Close is a no-op for the Azure client.
func (s *Store) Close() error {
	return nil
}

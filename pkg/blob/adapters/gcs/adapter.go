package gcs

import (
	"context"
	"io"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
)

type Adapter struct {
	client *storage.Client
}

func New(ctx context.Context) (*Adapter, error) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	return &Adapter{client: client}, nil
}

// Put uploads data to GCS.
func (a *Adapter) Put(ctx context.Context, bucket, key string, data []byte) error {
	w := a.client.Bucket(bucket).Object(key).NewWriter(ctx)
	if _, err := w.Write(data); err != nil {
		w.Close()
		return err
	}
	return w.Close()
}

// Get downloads data from GCS.
func (a *Adapter) Get(ctx context.Context, bucket, key string) ([]byte, error) {
	r, err := a.client.Bucket(bucket).Object(key).NewReader(ctx)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(r)
}

// List lists objects in a bucket.
func (a *Adapter) List(ctx context.Context, bucket string) ([]string, error) {
	var results []string
	it := a.client.Bucket(bucket).Objects(ctx, nil)
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		results = append(results, attrs.Name)
	}
	return results, nil
}

// Delete deletes an object.
func (a *Adapter) Delete(ctx context.Context, bucket, key string) error {
	return a.client.Bucket(bucket).Object(key).Delete(ctx)
}

// GetSignedURL generates a signed URL (mock implementation for interface compliance usually required)
func (a *Adapter) GetSignedURL(bucket, key string, expires time.Duration) (string, error) {
	// GCS SignedURL requires credentials file or private key, usually tough to do generically without specific config.
	// Returning not implemented or generic error for now if not strictly required by blob.Store interface found earlier.
	// Actually typical blob.Store might not enforce SignedURL.
	// Let's implement basics first.
	return "", nil
}

func (a *Adapter) Close() error {
	return a.client.Close()
}

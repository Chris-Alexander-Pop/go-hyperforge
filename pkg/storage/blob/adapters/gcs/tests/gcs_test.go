package gcs_test

import (
	"context"
	"os"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/storage/blob"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/storage/blob/adapters/gcs"
)

func TestGCSAdapter_Init(t *testing.T) {
	if os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") == "" {
		t.Skip("Skipping GCS test: GOOGLE_APPLICATION_CREDENTIALS not set")
	}

	store, err := gcs.New(context.Background(), blob.Config{Bucket: "test-bucket"})
	if err != nil {
		t.Logf("GCS New returned error: %v", err)
	} else if store == nil {
		t.Error("Returned nil store")
	}
}

func TestGCSAdapter_RequiresBucket(t *testing.T) {
	_, err := gcs.New(context.Background(), blob.Config{})
	if err == nil {
		t.Fatal("expected error for empty bucket")
	}
	if !blob.IsInvalidArgument(err) && err != blob.ErrInvalidConfig {
		// ErrInvalidConfig is InvalidArgument-coded sentinel
		if !blob.IsInvalidArgument(err) {
			t.Fatalf("expected invalid config, got %v", err)
		}
	}
}

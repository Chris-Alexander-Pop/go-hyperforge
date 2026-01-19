package gcs_test

import (
	"context"
	"os"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/storage/blob/adapters/gcs"
)

func TestGCSAdapter_Init(t *testing.T) {
	if os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") == "" {
		t.Skip("Skipping GCS test: GOOGLE_APPLICATION_CREDENTIALS not set")
	}

	store, err := gcs.New(context.Background())

	if err != nil {
		t.Logf("GCS New returned error: %v", err)
	} else if store == nil {
		t.Error("Returned nil store")
	}
}

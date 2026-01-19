package s3_test

import (
	"context"
	"os"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/storage/blob"
	"github.com/chris-alexander-pop/system-design-library/pkg/storage/blob/adapters/s3"
)

func TestS3Adapter_Init(t *testing.T) {
	// Basic initialization test only, as we don't mock full AWS SDK in this step
	if os.Getenv("AWS_REGION") == "" {
		t.Skip("Skipping S3 test: AWS_REGION not set")
	}

	// Just verify it compiles and creates (or fails cleanly)
	store, err := s3.New(context.Background(), blob.Config{
		Bucket: "test-bucket",
		Region: "us-east-1",
	})

	if err == nil {
		// Valid case if credentials exist
		if store == nil {
			t.Error("Returned nil store without error")
		}
	} else {
		// Also valid if credentials missing
		t.Logf("S3 New returned error (expected without creds): %v", err)
	}
}

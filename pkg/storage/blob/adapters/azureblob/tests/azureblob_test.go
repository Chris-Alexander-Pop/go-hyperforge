package azureblob_test

import (
	"os"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/storage/blob"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/storage/blob/adapters/azureblob"
)

func TestAzureAdapter_Init(t *testing.T) {
	if os.Getenv("AZURE_STORAGE_ACCOUNT") == "" {
		t.Skip("Skipping Azure test: AZURE_STORAGE_ACCOUNT not set")
	}

	store, err := azureblob.New(blob.Config{
		AzureAccountName: "testaccount",
		Bucket:           "test-container",
	})
	if err != nil {
		t.Logf("Azure New returned error: %v", err)
	} else if store == nil {
		t.Error("Returned nil store")
	}
}

func TestAzureAdapter_RequiresAccountAndBucket(t *testing.T) {
	_, err := azureblob.New(blob.Config{})
	if err == nil {
		t.Fatal("expected error for empty config")
	}
	if err != blob.ErrInvalidConfig && !blob.IsInvalidArgument(err) {
		t.Fatalf("expected invalid config, got %v", err)
	}
}

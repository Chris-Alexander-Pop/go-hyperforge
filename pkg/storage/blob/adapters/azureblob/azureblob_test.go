package azureblob_test

import (
	"os"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/storage/blob/adapters/azureblob"
)

func TestAzureAdapter_Init(t *testing.T) {
	if os.Getenv("AZURE_STORAGE_ACCOUNT") == "" {
		t.Skip("Skipping Azure test: AZURE_STORAGE_ACCOUNT not set")
	}

	// Assuming required config, just testing New signature call
	store, err := azureblob.New("test-account")

	if err != nil {
		t.Logf("Azure New returned error: %v", err)
	} else if store == nil {
		t.Error("Returned nil store")
	}
}

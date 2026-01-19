package eventhubs_test

import (
	"os"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/streaming/adapters/eventhubs"
)

func TestEventHubs_Init(t *testing.T) {
	if os.Getenv("AZURE_EVENTHUB_CONNECTION_STRING") == "" {
		t.Skip("Skipping EventHubs test")
	}

	client, err := eventhubs.New("test-ns", "test-hub")
	if err != nil {
		t.Logf("EventHubs New error: %v", err)
	} else if client == nil {
		t.Error("Returned nil client")
	}
}

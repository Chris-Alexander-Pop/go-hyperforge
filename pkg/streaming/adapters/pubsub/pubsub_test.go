package pubsub_test

import (
	"context"
	"os"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/streaming/adapters/pubsub"
)

func TestPubSub_Init(t *testing.T) {
	if os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") == "" {
		t.Skip("Skipping PubSub test")
	}

	client, err := pubsub.New(context.Background(), "test-project")
	if err != nil {
		t.Logf("PubSub New error: %v", err)
	} else if client == nil {
		t.Error("Returned nil client")
	}
}

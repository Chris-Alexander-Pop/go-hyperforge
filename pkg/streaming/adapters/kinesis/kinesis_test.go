package kinesis_test

import (
	"context"
	"os"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/streaming/adapters/kinesis"
)

func TestKinesis_Init(t *testing.T) {
	if os.Getenv("AWS_REGION") == "" {
		t.Skip("Skipping Kinesis test")
	}

	client, err := kinesis.New(context.Background())

	if err == nil {
		if client == nil {
			t.Error("Returned nil client")
		}
	} else {
		t.Logf("Kinesis New error (expected if no creds): %v", err)
	}
}

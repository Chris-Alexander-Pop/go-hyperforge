package redshift_test

import (
	"context"
	"os"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/bigdata/adapters/redshift"
)

func TestRedshift_Init(t *testing.T) {
	if os.Getenv("AWS_REGION") == "" {
		t.Skip("Skipping Redshift test")
	}

	adapter, err := redshift.New(context.Background(), "cluster-id", "db", "user")
	if err != nil {
		t.Logf("Redshift New error: %v", err)
	} else if adapter == nil {
		t.Error("Returned nil adapter")
	}
}

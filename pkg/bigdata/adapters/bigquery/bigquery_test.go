package bigquery_test

import (
	"context"
	"os"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/bigdata/adapters/bigquery"
)

func TestBigQuery_Init(t *testing.T) {
	if os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") == "" {
		t.Skip("Skipping BigQuery test")
	}

	adapter, err := bigquery.New(context.Background(), "test-project")
	if err != nil {
		t.Logf("BigQuery New error: %v", err)
	} else if adapter == nil {
		t.Error("Returned nil adapter")
	}
}

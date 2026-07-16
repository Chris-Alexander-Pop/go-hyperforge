package cosmosdb

import (
	"testing"
)

func TestCosmosDBNoSQLInjection_BuildQuery(t *testing.T) {
	query := map[string]interface{}{
		"id OR 1=1": "some-val",
	}

	_, _, err := buildQuery(query)
	if err == nil {
		t.Errorf("expected error for invalid query key, got nil")
	}
}

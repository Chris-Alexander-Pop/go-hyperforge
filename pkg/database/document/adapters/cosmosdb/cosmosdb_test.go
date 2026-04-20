package cosmosdb

import (
	"testing"
)

func TestBuildQuery_Injection(t *testing.T) {
	query := map[string]interface{}{
		"id = '1' OR 1=1": "admin",
	}

	_, _, err := buildQuery(query)
	if err == nil {
		t.Errorf("Expected error for invalid query key, got nil")
	}
}

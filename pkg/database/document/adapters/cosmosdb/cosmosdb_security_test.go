package cosmosdb

import (
	"testing"
)

func TestBuildQueryInjection(t *testing.T) {
	query := map[string]interface{}{
		"1=1 OR c.admin": true,
	}

	_, _, err := buildQuery(query)
	if err == nil {
		t.Fatal("Expected error for invalid query key, got nil")
	}
	t.Logf("Successfully blocked invalid query key: %v", err)
}

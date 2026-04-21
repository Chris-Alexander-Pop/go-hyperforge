package cosmosdb

import (
	"testing"
)

func TestBuildQuery_Injection(t *testing.T) {
	maliciousQuery := map[string]interface{}{"name = 'a' OR 1=1; --": "john"}
	_, _, err := buildQuery(maliciousQuery)
	if err == nil {
		t.Errorf("Expected error for invalid key, got nil")
	}
}

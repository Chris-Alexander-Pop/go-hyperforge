package cosmosdb

import (
	"fmt"
	"testing"
)

// Legacy implementation copy-pasted from Find for benchmarking
func buildQueryLegacy(query map[string]interface{}) string {
	queryText := "SELECT * FROM c"
	if len(query) > 0 {
		queryText += " WHERE "
		i := 0
		for k, v := range query {
			if i > 0 {
				queryText += " AND "
			}
			switch val := v.(type) {
			case string:
				queryText += fmt.Sprintf("c.%s = '%s'", k, val)
			default:
				queryText += fmt.Sprintf("c.%s = %v", k, v)
			}
			i++
		}
	}
	return queryText
}

func BenchmarkBuildQueryLegacy(b *testing.B) {
	query := map[string]interface{}{
		"name": "test",
		"age":  30,
		"city": "Seattle",
		"active": true,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buildQueryLegacy(query)
	}
}

func BenchmarkBuildQueryParameterized(b *testing.B) {
	query := map[string]interface{}{
		"name": "test",
		"age":  30,
		"city": "Seattle",
		"active": true,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buildQuery(query)
	}
}

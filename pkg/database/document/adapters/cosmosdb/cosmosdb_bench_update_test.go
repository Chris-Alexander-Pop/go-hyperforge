package cosmosdb

import (
	"fmt"
	"testing"
)

func BenchmarkUpdateFmtSprintf(b *testing.B) {
    update := map[string]interface{}{
		"field1": "value1",
		"field2": "value2",
		"field3": "value3",
		"field4": "value4",
		"field5": "value5",
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
        for k := range update {
            _ = fmt.Sprintf("/%s", k)
        }
	}
}

func BenchmarkUpdateStringConcat(b *testing.B) {
    update := map[string]interface{}{
		"field1": "value1",
		"field2": "value2",
		"field3": "value3",
		"field4": "value4",
		"field5": "value5",
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
        for k := range update {
            _ = "/" + k
        }
	}
}

package dynamodb

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// Benchmark using fmt.Sprintf
func BenchmarkQueryFmtSprintf(b *testing.B) {
	query := map[string]interface{}{
		"field1": "value1",
		"field2": 123,
		"field3": true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parts := make([]string, 0, len(query))
		expAttrValues := make(map[string]types.AttributeValue, len(query))
		expAttrNames := make(map[string]string, len(query))

		idx := 0
		for k := range query {
			placeholder := fmt.Sprintf(":v%d", idx)
			namePlaceholder := fmt.Sprintf("#n%d", idx)
			parts = append(parts, fmt.Sprintf("%s = %s", namePlaceholder, placeholder))
			expAttrNames[namePlaceholder] = k
			idx++
		}

		_ = parts
		_ = expAttrValues
		_ = expAttrNames
	}
}

// Benchmark using direct concatenation and strconv
func BenchmarkQueryStrconv(b *testing.B) {
	query := map[string]interface{}{
		"field1": "value1",
		"field2": 123,
		"field3": true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parts := make([]string, 0, len(query))
		expAttrValues := make(map[string]types.AttributeValue, len(query))
		expAttrNames := make(map[string]string, len(query))

		idx := 0
		for k := range query {
			idxStr := strconv.Itoa(idx)
			placeholder := ":v" + idxStr
			namePlaceholder := "#n" + idxStr
			parts = append(parts, namePlaceholder+" = "+placeholder)
			expAttrNames[namePlaceholder] = k
			idx++
		}

		_ = parts
		_ = expAttrValues
		_ = expAttrNames
	}
}

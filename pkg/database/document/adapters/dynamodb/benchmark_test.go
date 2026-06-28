package dynamodb

import (
	"fmt"
	"strings"
	"strconv"
	"testing"
)

func buildQueryLegacy(query map[string]interface{}) string {
    parts := []string{}
    i := 0
    for k := range query {
        placeholder := fmt.Sprintf(":v%d", i)
        namePlaceholder := fmt.Sprintf("#n%d", i)

        parts = append(parts, fmt.Sprintf("%s = %s", namePlaceholder, placeholder))
        _ = k // keeping k to avoid unused error
        i++
    }
    return strings.Join(parts, " AND ")
}

func buildQueryOptimized(query map[string]interface{}) string {
    if len(query) == 0 {
        return ""
    }

    // Pre-calculate estimated size to reduce re-allocations
    // Approximate 30 bytes per parameter (":v123", "#n123 = ")
    size := len(query) * 30

    var sb strings.Builder
    sb.Grow(size)

    i := 0
    for k := range query {
        if i > 0 {
            sb.WriteString(" AND ")
        }

        idxStr := strconv.Itoa(i)

        sb.WriteString("#n")
        sb.WriteString(idxStr)
        sb.WriteString(" = :v")
        sb.WriteString(idxStr)

        _ = k
        i++
    }
    return sb.String()
}

func BenchmarkBuildQueryLegacy(b *testing.B) {
	query := map[string]interface{}{
		"name":   "test",
		"age":    30,
		"city":   "Seattle",
		"active": true,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buildQueryLegacy(query)
	}
}

func BenchmarkBuildQueryOptimized(b *testing.B) {
	query := map[string]interface{}{
		"name":   "test",
		"age":    30,
		"city":   "Seattle",
		"active": true,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buildQueryOptimized(query)
	}
}

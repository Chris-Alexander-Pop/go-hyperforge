package fixedwindow_test

import (
	"fmt"
	"strconv"
	"testing"
)

// BenchmarkFixedWindow_KeyGeneration_FmtSprintf measures the performance of generating
// the cache key using fmt.Sprintf, the original implementation.
func BenchmarkFixedWindow_KeyGeneration_FmtSprintf(b *testing.B) {
	key := "my-api-key"
	window := int64(1700000000)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = fmt.Sprintf("rl:fixed:%s:%d", key, window)
	}
}

// BenchmarkFixedWindow_KeyGeneration_StrconvConcat measures the performance of generating
// the cache key using string concatenation and strconv.FormatInt, the optimized implementation.
func BenchmarkFixedWindow_KeyGeneration_StrconvConcat(b *testing.B) {
	key := "my-api-key"
	window := int64(1700000000)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = "rl:fixed:" + key + ":" + strconv.FormatInt(window, 10)
	}
}

package fixedwindow

import (
	"fmt"
	"strconv"
	"testing"
)

// BenchmarkSprintf simulates the old way of generating cache keys
func BenchmarkSprintf(b *testing.B) {
	key := "user_12345"
	window := int64(1678901234)
	for i := 0; i < b.N; i++ {
		_ = fmt.Sprintf("rl:fixed:%s:%d", key, window)
	}
}

// BenchmarkConcatStrconv simulates the optimized way of generating cache keys
func BenchmarkConcatStrconv(b *testing.B) {
	key := "user_12345"
	window := int64(1678901234)
	for i := 0; i < b.N; i++ {
		_ = "rl:fixed:" + key + ":" + strconv.FormatInt(window, 10)
	}
}

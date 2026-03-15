package fixedwindow

import (
	"fmt"
	"strconv"
	"testing"
	"time"
)

func BenchmarkFixedWindow_Sprintf(b *testing.B) {
	key := "test_key"
	window := time.Now().Unix()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = fmt.Sprintf("rl:fixed:%s:%d", key, window)
	}
}

func BenchmarkFixedWindow_ConcatFormatInt(b *testing.B) {
	key := "test_key"
	window := time.Now().Unix()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = "rl:fixed:" + key + ":" + strconv.FormatInt(window, 10)
	}
}

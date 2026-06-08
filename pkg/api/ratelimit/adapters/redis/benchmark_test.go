package redis

import (
	"fmt"
	"strconv"
	"testing"
	"time"
)

func BenchmarkSlidingWindowConcat(b *testing.B) {
	now := time.Now().UnixMilli()
	t_nano := time.Now().UnixNano()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Just the string operations we want to benchmark, to isolate the overhead
		_ = strconv.FormatInt(now, 10) + ":" + strconv.FormatInt(t_nano%1000000, 10)
		_ = "rl:dist:slide:" + "user-123"
	}
}

func BenchmarkSlidingWindowSprintf(b *testing.B) {
	now := time.Now().UnixMilli()
	t_nano := time.Now().UnixNano()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Just the string operations we want to benchmark
		_ = fmt.Sprintf("%d:%d", now, t_nano%1000000)
		_ = fmt.Sprintf("rl:dist:slide:%s", "user-123")
	}
}

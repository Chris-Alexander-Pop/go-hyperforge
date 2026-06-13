package redis

import (
	"fmt"
	"testing"
	"time"
	"strconv"
)

func BenchmarkKeyGenerationSprintf(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = fmt.Sprintf("rl:dist:fixed:%s", "test_key")
	}
}

func BenchmarkKeyGenerationConcat(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = "rl:dist:fixed:" + "test_key"
	}
}

func BenchmarkRequestIDSprintf(b *testing.B) {
	now := time.Now().UnixMilli()
	nanosec := time.Now().UnixNano()%1000000
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = fmt.Sprintf("%d:%d", now, nanosec)
	}
}

func BenchmarkRequestIDConcat(b *testing.B) {
	now := time.Now().UnixMilli()
	nanosec := time.Now().UnixNano()%1000000
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = strconv.FormatInt(now, 10) + ":" + strconv.FormatInt(nanosec, 10)
	}
}

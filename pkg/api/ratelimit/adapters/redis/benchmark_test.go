package redis

import (
	"fmt"
	"strconv"
	"testing"
	"time"
)

func BenchmarkFormat(b *testing.B) {
	key := "user_123456"
	now := time.Now().UnixMilli()
	nanosec := time.Now().UnixNano() % 1000000

	b.Run("Sprintf_Fixed", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = fmt.Sprintf("rl:dist:fixed:%s", key)
		}
	})

	b.Run("Concat_Fixed", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = "rl:dist:fixed:" + key
		}
	})

	b.Run("Sprintf_RequestID", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = fmt.Sprintf("%d:%d", now, nanosec)
		}
	})

	b.Run("Strconv_RequestID", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = strconv.FormatInt(now, 10) + ":" + strconv.FormatInt(nanosec, 10)
		}
	})
}

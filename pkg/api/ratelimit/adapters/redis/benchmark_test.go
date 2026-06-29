package redis

import (
	"fmt"
	"strconv"
	"testing"
	"time"
)

func BenchmarkKeyFmtSprintf(b *testing.B) {
	key := "user_12345"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = fmt.Sprintf("rl:dist:fixed:%s", key)
	}
}

func BenchmarkKeyConcat(b *testing.B) {
	key := "user_12345"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = "rl:dist:fixed:" + key
	}
}

func BenchmarkReqIDFmtSprintf(b *testing.B) {
	now := time.Now().UnixMilli()
	n := time.Now().UnixNano() % 1000000
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = fmt.Sprintf("%d:%d", now, n)
	}
}

func BenchmarkReqIDConcat(b *testing.B) {
	now := time.Now().UnixMilli()
	n := time.Now().UnixNano() % 1000000
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = strconv.FormatInt(now, 10) + ":" + strconv.FormatInt(n, 10)
	}
}

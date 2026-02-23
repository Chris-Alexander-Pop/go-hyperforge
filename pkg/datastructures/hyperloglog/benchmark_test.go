package hyperloglog_test

import (
	"crypto/rand"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/datastructures/hyperloglog"
)

func BenchmarkHyperLogLog_Add(b *testing.B) {
	hll := hyperloglog.New(14)
	data := make([]byte, 32)
	rand.Read(data)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		hll.Add(data)
	}
}

func BenchmarkHyperLogLog_AddString(b *testing.B) {
	hll := hyperloglog.New(14)
	// Create a 32-byte string to match data length
	str := "12345678901234567890123456789012"

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		hll.AddString(str)
	}
}

package hyperloglog_test

import (
	"crypto/rand"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/datastructures/hyperloglog"
)

func BenchmarkAdd(b *testing.B) {
	hll := hyperloglog.New(14)
	data := make([]byte, 8)
	rand.Read(data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Use a static buffer to avoid allocation noise in benchmark
		// or re-randomize if we want to test distribution (but static is better for pure function speed)
		hll.Add(data)
	}
}

func BenchmarkAddString(b *testing.B) {
	hll := hyperloglog.New(14)
	str := "test-string"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hll.AddString(str)
	}
}

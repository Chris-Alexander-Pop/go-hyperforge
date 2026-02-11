package hllpp

import (
	"crypto/rand"
	"testing"
)

func BenchmarkAdd(b *testing.B) {
	h := New(14)
	data := make([][]byte, b.N)
	for i := 0; i < b.N; i++ {
		data[i] = make([]byte, 8)
		rand.Read(data[i])
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.Add(data[i])
	}
}

func BenchmarkCount(b *testing.B) {
	h := New(14)
	for i := 0; i < 10000; i++ {
		d := make([]byte, 8)
		rand.Read(d)
		h.Add(d)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.Count()
	}
}

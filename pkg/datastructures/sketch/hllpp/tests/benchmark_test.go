package hllpp_test

import (
	"crypto/rand"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/datastructures/sketch/hllpp"
)

func BenchmarkAdd(b *testing.B) {
	h := hllpp.New(14)
	data := make([]byte, 32)
	rand.Read(data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.Add(data)
	}
}

func BenchmarkCount(b *testing.B) {
	h := hllpp.New(14)
	// Add some data to make it dense
	for i := 0; i < 20000; i++ {
		h.Add([]byte{byte(i), byte(i >> 8)})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.Count()
	}
}

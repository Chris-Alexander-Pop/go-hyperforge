package bloomfilter

import (
	"crypto/rand"
	"testing"
)

func BenchmarkDoubleHash(b *testing.B) {
	data := make([]byte, 32)
	rand.Read(data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		doubleHash(data)
	}
}

func BenchmarkBloomFilter_Add(b *testing.B) {
	bf := New(10000, 0.01)
	data := make([]byte, 32)
	rand.Read(data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bf.Add(data)
	}
}

func BenchmarkBloomFilter_Contains(b *testing.B) {
	bf := New(10000, 0.01)
	data := make([]byte, 32)
	rand.Read(data)
	bf.Add(data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bf.Contains(data)
	}
}

func BenchmarkBloomFilter_AddString(b *testing.B) {
	bf := New(10000, 0.01)
	str := "test-string-data"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bf.AddString(str)
	}
}

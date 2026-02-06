package bloomfilter_test

import (
	"crypto/rand"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/datastructures/bloomfilter"
)

func BenchmarkBloomFilter_Add(b *testing.B) {
	bf := bloomfilter.New(10000, 0.01)
	data := make([]byte, 32)
	rand.Read(data)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		bf.Add(data)
	}
}

func BenchmarkBloomFilter_Contains(b *testing.B) {
	bf := bloomfilter.New(10000, 0.01)
	data := make([]byte, 32)
	rand.Read(data)
	bf.Add(data)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		bf.Contains(data)
	}
}

func BenchmarkBloomFilter_AddString(b *testing.B) {
	bf := bloomfilter.New(10000, 0.01)
	str := "some-random-string-for-benchmarking"

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		bf.AddString(str)
	}
}

func BenchmarkBloomFilter_ContainsString(b *testing.B) {
	bf := bloomfilter.New(10000, 0.01)
	str := "some-random-string-for-benchmarking"
	bf.AddString(str)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		bf.ContainsString(str)
	}
}

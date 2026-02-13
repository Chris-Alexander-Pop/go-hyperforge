package hllpp_test

import (
	"fmt"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/datastructures/sketch/hllpp"
)

func BenchmarkHLLPP_Add(b *testing.B) {
	h := hllpp.New(14)
	data := make([]byte, 32)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data[0] = byte(i)
		data[1] = byte(i >> 8)
		data[2] = byte(i >> 16)
		h.Add(data)
	}
}

func BenchmarkHLLPP_Add_Dense(b *testing.B) {
	h := hllpp.New(14)
	for i := 0; i < 5000; i++ {
		h.Add([]byte(fmt.Sprintf("%d", i)))
	}
	if h.IsSparse {
		b.Fatal("Expected dense mode")
	}

	data := make([]byte, 32)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data[0] = byte(i)
		data[1] = byte(i >> 8)
		data[2] = byte(i >> 16)
		h.Add(data)
	}
}

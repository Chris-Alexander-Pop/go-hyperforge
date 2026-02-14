package hllpp

import (
	"fmt"
	"testing"
)

func BenchmarkAdd(b *testing.B) {
	h := New(14)
	data := []byte("test-data-string")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.Add(data)
	}
}

func BenchmarkAddStringConversion(b *testing.B) {
	h := New(14)
	str := "test-data-string"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.Add([]byte(str))
	}
}

func BenchmarkAddString(b *testing.B) {
	h := New(14)
	str := "test-data-string"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.AddString(str)
	}
}

func BenchmarkAddStringConversionDynamic(b *testing.B) {
	h := New(14)
	strs := make([]string, 1000)
	for i := range strs {
		strs[i] = fmt.Sprintf("test-data-%d", i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.Add([]byte(strs[i%1000]))
	}
}

func BenchmarkAddStringDynamic(b *testing.B) {
	h := New(14)
	strs := make([]string, 1000)
	for i := range strs {
		strs[i] = fmt.Sprintf("test-data-%d", i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.AddString(strs[i%1000])
	}
}

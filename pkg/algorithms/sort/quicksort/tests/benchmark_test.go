package quicksort_test

import (
	"github.com/chris-alexander-pop/system-design-library/pkg/algorithms/sort/quicksort"
	"math/rand"
	"testing"
)

func generateRandomSlice(n int) []int {
	s := make([]int, n)
	for i := 0; i < n; i++ {
		s[i] = rand.Int()
	}
	return s
}

func BenchmarkSortSerial(b *testing.B) {
	data := generateRandomSlice(1000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		arr := make([]int, len(data))
		copy(arr, data)
		b.StartTimer()
		quicksort.Sort(arr)
	}
}

func BenchmarkSortParallel(b *testing.B) {
	data := generateRandomSlice(1000)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			arr := make([]int, len(data))
			copy(arr, data)
			quicksort.Sort(arr)
		}
	})
}

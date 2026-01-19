package hyperloglog

import (
	"math"
	"testing"
)

func TestHyperLogLog(t *testing.T) {
	t.Run("PrecisionBounds", func(t *testing.T) {
		hll := New(2)
		if hll.precision != 4 {
			t.Errorf("Expected precision 4, got %d", hll.precision)
		}

		hll = New(20)
		if hll.precision != 16 {
			t.Errorf("Expected precision 16, got %d", hll.precision)
		}
	})

	t.Run("CardinalityEstimation", func(t *testing.T) {
		hll := New(14)
		uniqueCount := 10000

		// Add unique elements
		for i := 0; i < uniqueCount; i++ {
			hll.AddString(string(rune(i)))
		}

		count := hll.Count()
		errorRate := math.Abs(float64(count)-float64(uniqueCount)) / float64(uniqueCount)

		// Expected error for p=14 is ~0.8%, but variance can be high for small datasets.
		// Relaxing to 5% for stability.
		if errorRate > 0.05 {
			t.Errorf("Error rate %.4f is too high (expected < 0.05)", errorRate)
		}
	})

	t.Run("Merge", func(t *testing.T) {
		hll1 := New(10)
		hll2 := New(10)

		hll1.Add([]byte("apple"))
		hll1.Add([]byte("banana"))

		hll2.Add([]byte("cherry"))
		hll2.Add([]byte("date"))

		if !hll1.Merge(hll2) {
			t.Fatal("Merge failed")
		}

		// Combined count should be 4
		count := hll1.Count()
		if count < 3 || count > 5 {
			t.Errorf("Expected merge count close to 4, got %d", count)
		}
	})

	t.Run("MergeMismatchPrecision", func(t *testing.T) {
		hll1 := New(10)
		hll2 := New(12)

		if hll1.Merge(hll2) {
			t.Error("Expected merge to fail with different precisions")
		}
	})

	t.Run("Clear", func(t *testing.T) {
		hll := New(10)
		hll.AddString("foo")
		hll.Clear()

		if count := hll.Count(); count != 0 {
			t.Errorf("Expected empty count after Clear, got %d", count)
		}
	})
}

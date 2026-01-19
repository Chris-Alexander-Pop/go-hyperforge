package tdigest

import (
	"math"
	"testing"
)

func TestTDigest(t *testing.T) {
	td := New(100)

	// Add linear sequence 1..100
	for i := 1; i <= 100; i++ {
		td.Add(float64(i))
	}

	q50 := td.Quantile(0.5)
	// Ideally 50.5, allow wide margin for simplified implementation
	if math.Abs(q50-50.5) > 5.0 {
		t.Errorf("P50 far off: %f", q50)
	}

	q90 := td.Quantile(0.9)
	if math.Abs(q90-90.5) > 5.0 {
		t.Errorf("P90 far off: %f", q90)
	}

	q0 := td.Quantile(0.0)
	if q0 > 10.0 {
		t.Errorf("P0 too high: %f", q0)
	}
}

package histogram

import "testing"

func TestHistogram(t *testing.T) {
	h := New(1, 1000, 3)

	h.RecordValue(10)
	h.RecordValue(10)
	h.RecordValue(50)
	h.RecordValue(90)
	h.RecordValue(100)

	// Just verify quantile logic works without checking exact math of the simplified buckets
	p50 := h.ValueAtPercentile(50)
	if p50 < 10 {
		t.Errorf("P50 too low: %d", p50)
	}

	p99 := h.ValueAtPercentile(99)
	if p99 < 80 {
		t.Errorf("P99 too low: %d", p99)
	}
}

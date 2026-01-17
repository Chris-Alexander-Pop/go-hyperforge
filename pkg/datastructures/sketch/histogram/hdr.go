package histogram

import (
	"math"
)

// Histogram records values with high dynamic range and configurable precision.
type Histogram struct {
	counts      []int64
	minVal      int64
	maxVal      int64
	total       int64
	sigFigs     int
	bucketCount int
}

func New(minVal, maxVal int64, sigFigs int) *Histogram {
	// Simplified using linear + log buckets logic from HdrHistogram.
	// We'll just use a large float bucket array for demonstration of tracking.
	// Real Hdr uses clever bit indexing.

	// Fallback to simple Log-Linear bucket mapping.
	// size = log(range) * precision

	return &Histogram{
		counts:  make([]int64, 4096), // Fixed size for simplicity
		minVal:  minVal,
		maxVal:  maxVal,
		sigFigs: sigFigs,
	}
}

func (h *Histogram) RecordValue(val int64) {
	idx := h.getIndex(val)
	if idx >= 0 && idx < len(h.counts) {
		h.counts[idx]++
		h.total++
	}
}

func (h *Histogram) ValueAtPercentile(p float64) int64 {
	target := int64(float64(h.total) * (p / 100.0))
	var seen int64
	for i, c := range h.counts {
		seen += c
		if seen >= target {
			return h.getValue(i)
		}
	}
	return h.maxVal
}

func (h *Histogram) getIndex(val int64) int {
	// Log mapping
	if val <= 0 {
		return 0
	}
	// Simplistic log bucket
	return int(math.Log10(float64(val)) * float64(10*h.sigFigs))
}

func (h *Histogram) getValue(idx int) int64 {
	return int64(math.Pow(10, float64(idx)/float64(10*h.sigFigs)))
}

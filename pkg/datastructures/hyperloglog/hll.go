// Package hyperloglog provides probabilistic cardinality estimation.
//
// HyperLogLog estimates the number of distinct elements in a dataset
// using very little memory. Error rate is typically ±2% with 1KB of memory.
//
// Use cases:
//   - Counting unique visitors to a website
//   - Counting unique search queries
//   - Finding cardinality of large datasets
package hyperloglog

import (
	"hash/fnv"
	"math"
	"sync"
)

// HyperLogLog is a probabilistic cardinality estimator.
type HyperLogLog struct {
	registers []uint8 // Bucket registers
	precision uint8   // Precision (log2 of number of registers)
	numRegs   uint32  // Number of registers (2^precision)
	mu        sync.RWMutex
}

// New creates a new HyperLogLog with the given precision.
// Precision must be between 4 and 16 (inclusive).
// Higher precision = more accuracy but more memory.
//
// Memory usage: 2^precision bytes
// Typical error rate: 1.04 / sqrt(2^precision)
//
// Recommended:
//   - precision 10: 1KB memory, ~3.25% error
//   - precision 12: 4KB memory, ~1.625% error
//   - precision 14: 16KB memory, ~0.8% error
func New(precision uint8) *HyperLogLog {
	if precision < 4 {
		precision = 4
	}
	if precision > 16 {
		precision = 16
	}

	numRegs := uint32(1) << precision

	return &HyperLogLog{
		registers: make([]uint8, numRegs),
		precision: precision,
		numRegs:   numRegs,
	}
}

// Add adds an element to the estimator.
func (hll *HyperLogLog) Add(data []byte) {
	hash := hashBytes(data)
	hll.addHash(hash)
}

// AddString adds a string to the estimator.
func (hll *HyperLogLog) AddString(s string) {
	hll.Add([]byte(s))
}

func (hll *HyperLogLog) addHash(hash uint64) {
	hll.mu.Lock()
	defer hll.mu.Unlock()

	// Use first 'precision' bits as register index
	regIdx := uint32(hash >> (64 - hll.precision))

	// Count leading zeros in remaining bits + 1
	remaining := (hash << hll.precision) | (1 << (hll.precision - 1))
	leadingZeros := countLeadingZeros(remaining) + 1

	// Update register if new value is larger
	if leadingZeros > hll.registers[regIdx] {
		hll.registers[regIdx] = leadingZeros
	}
}

// Count returns the estimated cardinality.
func (hll *HyperLogLog) Count() uint64 {
	hll.mu.RLock()
	defer hll.mu.RUnlock()

	// Calculate harmonic mean
	var sum float64
	zeroCount := 0

	for _, val := range hll.registers {
		sum += math.Pow(2, -float64(val))
		if val == 0 {
			zeroCount++
		}
	}

	m := float64(hll.numRegs)

	// Alpha correction factor
	alpha := getAlpha(hll.precision)

	// Raw estimate
	estimate := alpha * m * m / sum

	// Apply corrections for small and large ranges
	if estimate <= 2.5*m && zeroCount > 0 {
		// Linear counting for small cardinalities
		estimate = m * math.Log(m/float64(zeroCount))
	} else if estimate > (1.0/30.0)*(1<<32) {
		// Large range correction
		estimate = -(1 << 32) * math.Log(1-estimate/(1<<32))
	}

	return uint64(estimate)
}

// Merge combines another HyperLogLog into this one.
// Both must have the same precision.
func (hll *HyperLogLog) Merge(other *HyperLogLog) bool {
	if hll.precision != other.precision {
		return false
	}

	hll.mu.Lock()
	other.mu.RLock()
	defer hll.mu.Unlock()
	defer other.mu.RUnlock()

	for i := range hll.registers {
		if other.registers[i] > hll.registers[i] {
			hll.registers[i] = other.registers[i]
		}
	}

	return true
}

// Clear resets the estimator.
func (hll *HyperLogLog) Clear() {
	hll.mu.Lock()
	defer hll.mu.Unlock()
	for i := range hll.registers {
		hll.registers[i] = 0
	}
}

// hashBytes computes a 64-bit hash of the data.
func hashBytes(data []byte) uint64 {
	h := fnv.New64a()
	h.Write(data)
	return h.Sum64()
}

// countLeadingZeros counts leading zeros in a 64-bit integer.
func countLeadingZeros(x uint64) uint8 {
	if x == 0 {
		return 64
	}

	var count uint8
	for x&(1<<63) == 0 {
		count++
		x <<= 1
	}
	return count
}

// getAlpha returns the correction factor for the given precision.
func getAlpha(precision uint8) float64 {
	switch precision {
	case 4:
		return 0.673
	case 5:
		return 0.697
	case 6:
		return 0.709
	default:
		m := float64(uint32(1) << precision)
		return 0.7213 / (1 + 1.079/m)
	}
}

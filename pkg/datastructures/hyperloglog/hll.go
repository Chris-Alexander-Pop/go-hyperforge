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
	"math"
	"math/bits"

	"github.com/chris-alexander-pop/system-design-library/pkg/concurrency"
)

// HyperLogLog is a probabilistic cardinality estimator.
type HyperLogLog struct {
	registers []uint8 // Bucket registers
	Precision uint8   // Precision (log2 of number of registers)
	numRegs   uint32  // Number of registers (2^Precision)
	mu        *concurrency.SmartRWMutex
}

// New creates a new HyperLogLog with the given Precision.
// Precision must be between 4 and 16 (inclusive).
// Higher Precision = more accuracy but more memory.
//
// Memory usage: 2^Precision bytes
// Typical error rate: 1.04 / sqrt(2^Precision)
//
// Recommended:
//   - Precision 10: 1KB memory, ~3.25% error
//   - Precision 12: 4KB memory, ~1.625% error
//   - Precision 14: 16KB memory, ~0.8% error
func New(Precision uint8) *HyperLogLog {
	if Precision < 4 {
		Precision = 4
	}
	if Precision > 16 {
		Precision = 16
	}

	numRegs := uint32(1) << Precision

	return &HyperLogLog{
		registers: make([]uint8, numRegs),
		Precision: Precision,
		numRegs:   numRegs,
		mu:        concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "HyperLogLog"}),
	}
}

// Add adds an element to the estimator.
func (hll *HyperLogLog) Add(data []byte) {
	hash := hashBytes(data)
	hll.addHash(hash)
}

// AddString adds a string to the estimator.
func (hll *HyperLogLog) AddString(s string) {
	hash := hashString(s)
	hll.addHash(hash)
}

func (hll *HyperLogLog) addHash(hash uint64) {
	hll.mu.Lock()
	defer hll.mu.Unlock()

	// Use first 'Precision' bits as register index
	regIdx := uint32(hash >> (64 - hll.Precision))

	// Count leading zeros in remaining bits + 1
	remaining := (hash << hll.Precision) | (1 << (hll.Precision - 1))
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
	alpha := getAlpha(hll.Precision)

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
// Both must have the same Precision.
func (hll *HyperLogLog) Merge(other *HyperLogLog) bool {
	if hll.Precision != other.Precision {
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

// hashBytes computes a 64-bit hash of the data using inline FNV-1a.
func hashBytes(data []byte) uint64 {
	const offset64 = 14695981039346656037
	const prime64 = 1099511628211

	hash := uint64(offset64)
	for _, b := range data {
		hash ^= uint64(b)
		hash *= prime64
	}
	return mix(hash)
}

// hashString computes a 64-bit hash of the string using inline FNV-1a.
func hashString(s string) uint64 {
	const offset64 = 14695981039346656037
	const prime64 = 1099511628211

	hash := uint64(offset64)
	for i := 0; i < len(s); i++ {
		hash ^= uint64(s[i])
		hash *= prime64
	}
	return mix(hash)
}

func mix(h uint64) uint64 {
	h ^= h >> 33
	h *= 0xff51afd7ed558ccd
	h ^= h >> 33
	h *= 0xc4ceb9fe1a85ec53
	h ^= h >> 33
	return h
}

// countLeadingZeros counts leading zeros in a 64-bit integer.
func countLeadingZeros(x uint64) uint8 {
	return uint8(bits.LeadingZeros64(x))
}

// getAlpha returns the correction factor for the given Precision.
func getAlpha(Precision uint8) float64 {
	switch Precision {
	case 4:
		return 0.673
	case 5:
		return 0.697
	case 6:
		return 0.709
	default:
		m := float64(uint32(1) << Precision)
		return 0.7213 / (1 + 1.079/m)
	}
}

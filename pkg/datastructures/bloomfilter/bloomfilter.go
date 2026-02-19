// Package bloomfilter provides a probabilistic data structure for set membership testing.
//
// A Bloom filter can tell you:
//   - "Definitely not in set" (100% accurate)
//   - "Probably in set" (false positive rate configurable)
//
// Use cases:
//   - Avoid expensive database lookups for non-existent keys
//   - Caching layer to skip cache misses
//   - Deduplication in streaming systems
package bloomfilter

import (
	"hash"
	"math"
	"math/bits"
	"unsafe"

	"github.com/chris-alexander-pop/system-design-library/pkg/concurrency"
)

// BloomFilter is a space-efficient probabilistic data structure.
type BloomFilter struct {
	bits    []uint64 // Bit array
	numBits uint     // Total number of bits
	numHash uint     // Number of hash functions
	mu      *concurrency.SmartRWMutex
	count   uint64 // Approximate number of elements added
}

// New creates a new Bloom filter.
//
// Parameters:
//   - expectedElements: Estimated number of elements to add
//   - falsePositiveRate: Target false positive probability (e.g., 0.01 for 1%)
func New(expectedElements uint, falsePositiveRate float64) *BloomFilter {
	if expectedElements == 0 {
		expectedElements = 100
	}
	if falsePositiveRate <= 0 || falsePositiveRate >= 1 {
		falsePositiveRate = 0.01
	}

	// Calculate optimal size: m = -n * ln(p) / (ln(2)^2)
	numBits := uint(math.Ceil(-float64(expectedElements) * math.Log(falsePositiveRate) / (math.Ln2 * math.Ln2)))

	// Calculate optimal number of hash functions: k = m/n * ln(2)
	numHash := uint(math.Ceil(float64(numBits) / float64(expectedElements) * math.Ln2))
	if numHash < 1 {
		numHash = 1
	}

	// Round up to 64-bit boundary
	numWords := (numBits + 63) / 64

	return &BloomFilter{
		bits:    make([]uint64, numWords),
		numBits: numBits,
		numHash: numHash,
		mu:      concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "BloomFilter"}),
	}
}

// NewWithSize creates a Bloom filter with specific size parameters.
func NewWithSize(numBits, numHash uint) *BloomFilter {
	numWords := (numBits + 63) / 64
	return &BloomFilter{
		bits:    make([]uint64, numWords),
		numBits: numBits,
		numHash: numHash,
		mu:      concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "BloomFilter"}),
	}
}

// Add adds an element to the filter.
func (bf *BloomFilter) Add(data []byte) {
	// Calculate hash outside the lock to minimize contention and duration
	h1, h2 := doubleHash(data)

	bf.mu.Lock()
	defer bf.mu.Unlock()

	numBits := bf.numBits
	// Inline calculation of positions to avoid slice allocation
	for i := uint(0); i < bf.numHash; i++ {
		// h(i) = h1 + i*h2 (mod numBits)
		pos := (uint(h1) + i*uint(h2)) % numBits
		wordIdx := pos / 64
		bitIdx := pos % 64
		bf.bits[wordIdx] |= 1 << bitIdx
	}
	bf.count++
}

// AddString adds a string to the filter.
func (bf *BloomFilter) AddString(s string) {
	bf.Add(stringToBytes(s))
}

// Contains tests if an element might be in the filter.
// Returns false if definitely not in set, true if probably in set.
func (bf *BloomFilter) Contains(data []byte) bool {
	// Calculate hash outside the lock
	h1, h2 := doubleHash(data)

	bf.mu.RLock()
	defer bf.mu.RUnlock()

	numBits := bf.numBits
	for i := uint(0); i < bf.numHash; i++ {
		// h(i) = h1 + i*h2 (mod numBits)
		pos := (uint(h1) + i*uint(h2)) % numBits
		wordIdx := pos / 64
		bitIdx := pos % 64
		if bf.bits[wordIdx]&(1<<bitIdx) == 0 {
			return false
		}
	}
	return true
}

// ContainsString tests if a string might be in the filter.
func (bf *BloomFilter) ContainsString(s string) bool {
	return bf.Contains(stringToBytes(s))
}

// EstimatedFalsePositiveRate returns the current estimated false positive rate.
func (bf *BloomFilter) EstimatedFalsePositiveRate() float64 {
	bf.mu.RLock()
	defer bf.mu.RUnlock()

	// FPR ≈ (1 - e^(-k*n/m))^k
	m := float64(bf.numBits)
	k := float64(bf.numHash)
	n := float64(bf.count)

	return math.Pow(1-math.Exp(-k*n/m), k)
}

// Count returns the approximate number of elements added.
func (bf *BloomFilter) Count() uint64 {
	bf.mu.RLock()
	defer bf.mu.RUnlock()
	return bf.count
}

// Clear resets the filter.
func (bf *BloomFilter) Clear() {
	bf.mu.Lock()
	defer bf.mu.Unlock()
	for i := range bf.bits {
		bf.bits[i] = 0
	}
	bf.count = 0
}

// doubleHash computes two 64-bit hash values for double hashing.
// It uses an inline implementation of FNV-1a 128-bit to avoid allocations.
func doubleHash(data []byte) (uint64, uint64) {
	// Initialize with offset basis for FNV-1a 128-bit
	// Upper 64 bits: 0x6c62272e07bb0142
	// Lower 64 bits: 0x62b821756295c58d
	h1 := uint64(0x6c62272e07bb0142)
	h2 := uint64(0x62b821756295c58d)

	for _, b := range data {
		h2 ^= uint64(b)

		// Multiply by FNV prime (2^88 + 315)
		// p1 = 1<<24 (upper 64 bits of prime)
		// p2 = 315 (lower 64 bits of prime)

		// h2 * p2
		hi, lo := bits.Mul64(h2, 315)

		// h1 * p2 + h2 * p1 + carry
		// h2 * p1 is h2 << 24
		h1 = h1*315 + (h2 << 24) + hi
		h2 = lo
	}

	return h1, h2
}

// stringToBytes converts a string to a byte slice without allocation.
// The returned byte slice must not be modified.
func stringToBytes(s string) []byte {
	return unsafe.Slice(unsafe.StringData(s), len(s))
}

// Union merges another Bloom filter into this one.
// Both filters must have the same size and number of hash functions.
func (bf *BloomFilter) Union(other *BloomFilter) bool {
	if bf.numBits != other.numBits || bf.numHash != other.numHash {
		return false
	}

	bf.mu.Lock()
	other.mu.RLock()
	defer bf.mu.Unlock()
	defer other.mu.RUnlock()

	for i := range bf.bits {
		bf.bits[i] |= other.bits[i]
	}

	return true
}

// Helper for custom hash functions
type HashFactory func() hash.Hash64

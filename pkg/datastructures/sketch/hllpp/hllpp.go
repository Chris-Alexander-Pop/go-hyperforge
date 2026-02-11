package hllpp

import (
	"math"
	"math/bits"
)

// HLLPP is a simplified HyperLogLog++ implementation.
// It switches from sparse (linear counting) to dense (HLL) estimation.
type HLLPP struct {
	m         uint // number of registers
	p         uint // precision (log2 m)
	registers []uint8
	sparse    map[uint64]struct{}
	IsSparse  bool
	threshold uint
}

func New(p uint8) *HLLPP {
	if p < 4 {
		p = 4
	}
	if p > 16 {
		p = 16
	}
	m := uint(1 << p)
	return &HLLPP{
		m:         m,
		p:         uint(p),
		registers: make([]uint8, m),
		sparse:    make(map[uint64]struct{}),
		IsSparse:  true,
		threshold: m / 4, // Switch when sparse set has m/4 items (approx)
	}
}

func (h *HLLPP) Add(data []byte) {
	hash := hash64(data)
	if h.IsSparse {
		h.sparse[hash] = struct{}{}
		if uint(len(h.sparse)) > h.threshold {
			h.mergeSparse()
			h.IsSparse = false
		}
		return
	}

	// Dense Mode (Standard HLL)
	// Standard HLL uses p bits for index: w = hash >> (64-p)
	idx := hash >> (64 - h.p)
	val := hash << h.p // remaining bits
	rank := uint8(clz(val)) + 1

	if rank > h.registers[idx] {
		h.registers[idx] = rank
	}
}

func (h *HLLPP) mergeSparse() {
	for hash := range h.sparse {
		// In a real HLL++, we would store 64-bit integers in the sparse list.
		// Now we do.

		idx := hash >> (64 - h.p)
		val := hash << h.p // remaining bits
		rank := uint8(clz(val)) + 1

		if rank > h.registers[idx] {
			h.registers[idx] = rank
		}
	}
	// Clear sparse
	h.sparse = nil
}

func (h *HLLPP) Count() uint64 {
	if h.IsSparse {
		return uint64(len(h.sparse)) // Linear counting
	}

	// HLL Estimate
	alpha := 0.7213 / (1 + 1.079/float64(h.m))
	sum := 0.0
	for _, val := range h.registers {
		sum += math.Pow(2, -float64(val))
	}

	est := alpha * float64(h.m*h.m) / sum

	// Small range correction
	if est <= 2.5*float64(h.m) {
		zeros := 0
		for _, v := range h.registers {
			if v == 0 {
				zeros++
			}
		}
		if zeros > 0 {
			est = float64(h.m) * math.Log(float64(h.m)/float64(zeros))
		}
	}

	return uint64(est)
}

func hash64(data []byte) uint64 {
	// Inline FNV-1a to avoid allocation
	const offset64 = 14695981039346656037
	const prime64 = 1099511628211
	h := uint64(offset64)
	for _, b := range data {
		h ^= uint64(b)
		h *= prime64
	}
	return h
}

func clz(x uint64) int {
	return bits.LeadingZeros64(x)
}

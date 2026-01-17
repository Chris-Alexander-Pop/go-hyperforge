package scalable

import (
	"hash/fnv"
	"math"
)

// Filter grows dynamically by adding layers.
type Filter struct {
	filters    []*bloom
	r          float64 // sizing ratio
	fpRate     float64
	totalCount uint64
}

func New(initialCap uint, fpRate float64) *Filter {
	return &Filter{
		filters: []*bloom{newBloom(initialCap, fpRate)},
		r:       2.0, // Double size each layer
		fpRate:  fpRate,
	}
}

func (s *Filter) Add(data []byte) {
	if s.Contains(data) {
		return
	}

	last := s.filters[len(s.filters)-1]
	if last.count >= last.cap {
		// Add new layer
		newCap := uint(float64(last.cap) * s.r)
		// tighter fp rate for subsequent layers to maintain total probability?
		// P_total = 1 - product(1 - Pi).
		// Simplified: keep same rate or partial scaling.
		s.filters = append(s.filters, newBloom(newCap, s.fpRate*0.9)) // tightening
		last = s.filters[len(s.filters)-1]
	}

	last.add(data)
	s.totalCount++
}

func (s *Filter) Contains(data []byte) bool {
	for i := len(s.filters) - 1; i >= 0; i-- {
		if s.filters[i].contains(data) {
			return true
		}
	}
	return false
}

// simple bloom
type bloom struct {
	bits  []bool
	k     uint
	cap   uint
	count uint
}

func newBloom(cap uint, p float64) *bloom {
	m := uint(math.Ceil(-(float64(cap) * math.Log(p)) / (math.Log(2) * math.Log(2))))
	k := uint(math.Ceil((float64(m) / float64(cap)) * math.Log(2)))
	return &bloom{
		bits: make([]bool, m),
		k:    k,
		cap:  cap,
	}
}

func (b *bloom) add(data []byte) {
	h1, h2 := hash(data)
	for i := uint(0); i < b.k; i++ {
		idx := (h1 + uint64(i)*h2) % uint64(len(b.bits))
		b.bits[idx] = true
	}
	b.count++
}

func (b *bloom) contains(data []byte) bool {
	h1, h2 := hash(data)
	for i := uint(0); i < b.k; i++ {
		idx := (h1 + uint64(i)*h2) % uint64(len(b.bits))
		if !b.bits[idx] {
			return false
		}
	}
	return true
}

func hash(data []byte) (uint64, uint64) {
	h := fnv.New64a()
	h.Write(data)
	v1 := h.Sum64()
	h.Write([]byte{0})
	v2 := h.Sum64()
	return v1, v2
}

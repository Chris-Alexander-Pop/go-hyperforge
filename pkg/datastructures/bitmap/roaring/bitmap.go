package roaring

import (
	"sort"
)

// ContainerType represents the type of container.
type ContainerType int

const (
	ArrayContainer ContainerType = iota
	BitmapContainer
)

const (
	maxArraySize = 4096
)

// Bitmap is a simplified Roaring Bitmap implementation.
// It uses a two-level index: high 16 bits -> container.
// Containers are either Array (sparse) or Bitmap (dense).
type Bitmap struct {
	containers map[uint16]container
}

type container interface {
	add(val uint16) container
	contains(val uint16) bool
	startValues() []uint16 // Helper for iteration/conversion
}

// New creates a new Bitmap.
func New() *Bitmap {
	return &Bitmap{
		containers: make(map[uint16]container),
	}
}

// Add adds a value to the bitmap.
func (b *Bitmap) Add(x uint32) {
	hb := uint16(x >> 16)
	lb := uint16(x & 0xFFFF)

	c, exists := b.containers[hb]
	if !exists {
		c = newArrayContainer()
	}
	b.containers[hb] = c.add(lb)
}

// Contains checks if a value exists.
func (b *Bitmap) Contains(x uint32) bool {
	hb := uint16(x >> 16)
	lb := uint16(x & 0xFFFF)
	c, exists := b.containers[hb]
	return exists && c.contains(lb)
}

// arrayContainer holds sorted list of uint16.
type arrayContainer struct {
	content []uint16
}

func newArrayContainer() *arrayContainer {
	return &arrayContainer{content: make([]uint16, 0)}
}

func (ac *arrayContainer) add(val uint16) container {
	// Binary search to check existence
	idx := sort.Search(len(ac.content), func(i int) bool { return ac.content[i] >= val })
	if idx < len(ac.content) && ac.content[idx] == val {
		return ac
	}

	// Insert
	ac.content = append(ac.content, 0)
	copy(ac.content[idx+1:], ac.content[idx:])
	ac.content[idx] = val

	if len(ac.content) > maxArraySize {
		return ac.toBitmap()
	}
	return ac
}

func (ac *arrayContainer) contains(val uint16) bool {
	idx := sort.Search(len(ac.content), func(i int) bool { return ac.content[i] >= val })
	return idx < len(ac.content) && ac.content[idx] == val
}

func (ac *arrayContainer) toBitmap() *bitmapContainer {
	bc := newBitmapContainer()
	for _, v := range ac.content {
		bc.add(v)
	}
	return bc
}

func (ac *arrayContainer) startValues() []uint16 {
	return ac.content
}

// bitmapContainer holds 1024 uint64s (65536 bits).
type bitmapContainer struct {
	bitmap []uint64
}

func newBitmapContainer() *bitmapContainer {
	return &bitmapContainer{bitmap: make([]uint64, 1024)}
}

func (bc *bitmapContainer) add(val uint16) container {
	idx := val / 64
	bit := val % 64
	bc.bitmap[idx] |= (1 << bit)
	return bc
}

func (bc *bitmapContainer) contains(val uint16) bool {
	idx := val / 64
	bit := val % 64
	return (bc.bitmap[idx] & (1 << bit)) != 0
}

func (bc *bitmapContainer) startValues() []uint16 {
	// inefficient but interface compliant
	var res []uint16
	for i, word := range bc.bitmap {
		if word == 0 {
			continue
		}
		for j := 0; j < 64; j++ {
			if (word & (1 << j)) != 0 {
				res = append(res, uint16(i*64+j))
			}
		}
	}
	return res
}

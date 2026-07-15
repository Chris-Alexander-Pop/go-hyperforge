// Package typesense provides a memory-backed SearchEngine stub labeled for Typesense.
//
// This is NOT a Typesense HTTP/protocol client. It reuses the in-memory search
// engine for local tests and scaffolding until a real Typesense adapter lands.
package typesense

import (
	"github.com/chris-alexander-pop/go-hyperforge/pkg/data/search"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/data/search/adapters/memory"
)

// New returns a memory-backed SearchEngine suitable for Typesense-shaped local use.
func New() search.SearchEngine {
	return memory.New()
}

// NewWithConfig ignores cfg and returns a memory-backed engine.
func NewWithConfig(_ search.Config) search.SearchEngine {
	return memory.New()
}

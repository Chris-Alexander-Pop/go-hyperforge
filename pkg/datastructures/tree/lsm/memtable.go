package lsm

import (
	"sync"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/datastructures/skiplist"
)

// MemTable is an in-memory table sorted by key (using a SkipList).
// In a real LSM, this would flush to SSTables on disk when full.
type MemTable struct {
	sl   *skiplist.SkipList[string, []byte]
	size int64
	cap  int64
	mu   sync.RWMutex
}

func New(capacityBytes int64) *MemTable {
	return &MemTable{
		sl:  skiplist.New[string, []byte](),
		cap: capacityBytes,
	}
}

func (m *MemTable) Put(key string, value []byte) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	estimatedSize := int64(len(key) + len(value))
	if m.size+estimatedSize > m.cap {
		return false // Full, trigger flush
	}

	m.sl.Set(key, value)
	m.size += estimatedSize
	return true
}

func (m *MemTable) Get(key string) ([]byte, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.sl.Get(key)
}

func (m *MemTable) Delete(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	// In LSM, delete is a Tombstone insert.
	// For MemTable, we can just Set(key, tombstone) or delete if we only model one layer.
	// Let's model simple deletion for this interface.
	m.sl.Delete(key)
}

func (m *MemTable) Size() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.size
}

func (m *MemTable) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sl = skiplist.New[string, []byte]()
	m.size = 0
}

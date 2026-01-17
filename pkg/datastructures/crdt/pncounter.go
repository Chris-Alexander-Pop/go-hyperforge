package crdt

// PNCounter is a Positive-Negative Counter CRDT.
// It uses two GCounter instances: one for increments (P) and one for decrements (N).
type PNCounter struct {
	p *GCounter
	n *GCounter
}

func NewPNCounter(id string) *PNCounter {
	return &PNCounter{
		p: NewGCounter(id),
		n: NewGCounter(id),
	}
}

// Inc increments the counter.
func (c *PNCounter) Inc(delta uint64) {
	c.p.Inc(delta)
}

// Dec decrements the counter.
func (c *PNCounter) Dec(delta uint64) {
	c.n.Inc(delta)
}

// Count returns the current value.
func (c *PNCounter) Count() int64 {
	return int64(c.p.Count()) - int64(c.n.Count())
}

// Merge merges another PNCounter.
func (c *PNCounter) Merge(other *PNCounter) {
	c.p.Merge(other.p)
	c.n.Merge(other.n)
}

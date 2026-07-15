package weightedroundrobin

import (
	"context"
	"sync"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/algorithms/loadbalancing"
)

// Balancer selects nodes based on their weight using
// the "Interleaved Weighted Round Robin" algorithm.
type Balancer struct {
	nodes []*weightedNode
	gcd   int
	maxW  int
	i     int
	cw    int
	mu    sync.Mutex
}

type weightedNode struct {
	id     string
	weight int
}

// New creates a new WeightedRoundRobin balancer.
func New() *Balancer {
	return &Balancer{
		nodes: make([]*weightedNode, 0),
		i:     -1,
		cw:    0,
	}
}

// Next returns the next weighted node.
func (b *Balancer) Next(ctx context.Context) (string, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	n := len(b.nodes)
	if n == 0 {
		return "", loadbalancing.ErrNoNodes
	}

	for {
		b.i = (b.i + 1) % n
		if b.i == 0 {
			b.cw = b.cw - b.gcd
			if b.cw <= 0 {
				b.cw = b.maxW
				if b.cw == 0 {
					return "", loadbalancing.ErrNoNodes
				}
			}
		}

		if b.nodes[b.i].weight >= b.cw {
			return b.nodes[b.i].id, nil
		}
	}
}

// Add adds a weighted node.
func (b *Balancer) Add(node string, weight int) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if weight <= 0 {
		weight = 1
	}

	b.nodes = append(b.nodes, &weightedNode{id: node, weight: weight})
	b.recalc()
}

// Remove removes a node.
func (b *Balancer) Remove(node string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for idx, n := range b.nodes {
		if n.id == node {
			b.nodes = append(b.nodes[:idx], b.nodes[idx+1:]...)
			break
		}
	}
	b.recalc()
}

func (b *Balancer) recalc() {
	b.gcd = 0
	b.maxW = 0
	for _, n := range b.nodes {
		b.gcd = gcd(b.gcd, n.weight)
		if n.weight > b.maxW {
			b.maxW = n.weight
		}
	}
	b.i = -1
	b.cw = 0
}

func gcd(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

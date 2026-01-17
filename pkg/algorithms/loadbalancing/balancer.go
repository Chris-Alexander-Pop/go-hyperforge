package loadbalancing

import (
	"context"
	"errors"
	"math/rand"
	"sync"
	"sync/atomic"
)

var (
	ErrNoNodes = errors.New("no nodes available")
)

// Balancer chooses a node from a list of available nodes.
type Balancer interface {
	// Next returns the next node to use.
	Next(ctx context.Context) (string, error)
	// Add adds a node (with optional weight).
	Add(node string, weight int)
	// Remove removes a node.
	Remove(node string)
}

// -----------------------------------------------------------------------------
// Round Robin
// -----------------------------------------------------------------------------

// RoundRobin cycles through nodes sequentially.
type RoundRobin struct {
	nodes []string
	count uint64
	mu    sync.RWMutex
}

func NewRoundRobin(nodes ...string) *RoundRobin {
	return &RoundRobin{
		nodes: nodes,
	}
}

func (b *RoundRobin) Next(ctx context.Context) (string, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	n := len(b.nodes)
	if n == 0 {
		return "", ErrNoNodes
	}

	// atomic increment
	count := atomic.AddUint64(&b.count, 1)
	return b.nodes[(count-1)%uint64(n)], nil
}

func (b *RoundRobin) Add(node string, weight int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.nodes = append(b.nodes, node)
}

func (b *RoundRobin) Remove(node string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	// simple O(N) removal
	for i, n := range b.nodes {
		if n == node {
			b.nodes = append(b.nodes[:i], b.nodes[i+1:]...)
			return
		}
	}
}

// -----------------------------------------------------------------------------
// Random
// -----------------------------------------------------------------------------

// Random selects a node randomly.
type Random struct {
	nodes []string
	mu    sync.RWMutex
}

func NewRandom(nodes ...string) *Random {
	return &Random{
		nodes: nodes,
	}
}

func (b *Random) Next(ctx context.Context) (string, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	n := len(b.nodes)
	if n == 0 {
		return "", ErrNoNodes
	}

	return b.nodes[rand.Intn(n)], nil
}

func (b *Random) Add(node string, weight int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.nodes = append(b.nodes, node)
}

func (b *Random) Remove(node string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for i, n := range b.nodes {
		if n == node {
			b.nodes = append(b.nodes[:i], b.nodes[i+1:]...)
			return
		}
	}
}

// -----------------------------------------------------------------------------
// Least Connections
// -----------------------------------------------------------------------------

// LeastConnections selects the node with the fewest active connections.
// It requires manual instrumentation (Inc/Dec).
type LeastConnections struct {
	nodes map[string]int64 // node -> active count
	mu    sync.RWMutex
}

func NewLeastConnections(nodes ...string) *LeastConnections {
	m := make(map[string]int64)
	for _, n := range nodes {
		m[n] = 0
	}
	return &LeastConnections{
		nodes: m,
	}
}

func (b *LeastConnections) Next(ctx context.Context) (string, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if len(b.nodes) == 0 {
		return "", ErrNoNodes
	}

	var bestNode string
	var minConns int64 = -1

	for node, conns := range b.nodes {
		if minConns == -1 || conns < minConns {
			minConns = conns
			bestNode = node
		}
	}

	return bestNode, nil
}

// Inc increments the connection count for a node.
// Call this when a request starts.
func (b *LeastConnections) Inc(node string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, ok := b.nodes[node]; ok {
		b.nodes[node]++
	}
}

// Dec decrements the connection count for a node.
// Call this when a request ends.
func (b *LeastConnections) Dec(node string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if count, ok := b.nodes[node]; ok && count > 0 {
		b.nodes[node]--
	}
}

func (b *LeastConnections) Add(node string, weight int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, ok := b.nodes[node]; !ok {
		b.nodes[node] = 0
	}
}

func (b *LeastConnections) Remove(node string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.nodes, node)
}

// -----------------------------------------------------------------------------
// Weighted Round Robin
// -----------------------------------------------------------------------------

// WeightedRoundRobin selects nodes based on their weight using
// the "Interleaved Weighted Round Robin" algorithm (efficient for small weights).
type WeightedRoundRobin struct {
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

func NewWeightedRoundRobin() *WeightedRoundRobin {
	return &WeightedRoundRobin{
		nodes: make([]*weightedNode, 0),
	}
}

func (b *WeightedRoundRobin) Next(ctx context.Context) (string, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	n := len(b.nodes)
	if n == 0 {
		return "", ErrNoNodes
	}

	for {
		b.i = (b.i + 1) % n
		if b.i == 0 {
			b.cw = b.cw - b.gcd
			if b.cw <= 0 {
				b.cw = b.maxW
				if b.cw == 0 {
					return "", ErrNoNodes
				}
			}
		}

		if b.nodes[b.i].weight >= b.cw {
			return b.nodes[b.i].id, nil
		}
	}
}

func (b *WeightedRoundRobin) Add(node string, weight int) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if weight <= 0 {
		weight = 1
	}

	b.nodes = append(b.nodes, &weightedNode{id: node, weight: weight})
	b.recalc()
}

func (b *WeightedRoundRobin) Remove(node string) {
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

func (b *WeightedRoundRobin) recalc() {
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

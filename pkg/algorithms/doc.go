/*
Package algorithms provides implementations of common algorithms for distributed
systems, graph processing, search, sorting, and rate limiting.

Inventory (import paths under github.com/chris-alexander-pop/go-hyperforge):

	Search
	  - algorithms/search/binarysearch — generic binary search (Search, LowerBound, UpperBound)
	  - algorithms/search/ahocorasick — multi-pattern string matching

	Graph
	  - algorithms/graph — shared AdjacencyList / Weighted types
	  - algorithms/graph/bfs — breadth-first traversal and unweighted shortest path
	  - algorithms/graph/dfs — depth-first traversal and cycle detection
	  - algorithms/graph/dijkstra — weighted shortest path (uses pkg/datastructures/heap)
	  - algorithms/graph/astar — A* search (uses pkg/datastructures/heap)
	  - algorithms/graph/prim — MST (uses pkg/datastructures/heap); kruskal — MST
	  - algorithms/graph/louvain — educational Louvain first-phase (real ΔQ; not production)

	Sort
	  - algorithms/sort/quicksort, mergesort, heapsort, radixsort

	Rate limiting
	  - algorithms/ratelimit/tokenbucket — Local + DistLimiter (cache-backed)
	  - algorithms/ratelimit/slidingwindow — Local log + sliding-window counter
	  - algorithms/ratelimit/fixedwindow, leakybucket, htb, shaper

	Load balancing
	  - algorithms/loadbalancing/roundrobin, weightedroundrobin, leastconnections, random
	  - algorithms/loadbalancing/maglev, p2c, sticky, healthaware

	Consistent hashing / DHT
	  - algorithms/consistenthash/ring, bounded
	  - algorithms/dht/chord — educational Chord (Join/Stabilize/Notify + in-process Transport)

	Consensus / gossip
	  - algorithms/consensus/raft — educational Raft (log append/replicate)
	  - algorithms/consensus/paxos — educational Paxos (Learner + Multi-Paxos slots)
	  - algorithms/gossip/swim — educational SWIM (Events/Stop/incarnation refute)

	Concurrency helpers
	  - algorithms/concurrency/adaptive
*/
package algorithms

package graph

// AdjacencyList is an unweighted directed (or undirected-as-bidirectional) graph
// mapping each node ID to its neighbor IDs.
type AdjacencyList map[string][]string

// Weighted maps each node to neighbor → edge weight.
type Weighted map[string]map[string]float64

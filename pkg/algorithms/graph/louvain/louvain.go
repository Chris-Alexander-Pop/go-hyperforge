package louvain

// Community represents a set of nodes.
type Community struct {
	Nodes []int
}

// Graph is simple adjacency list for node IDs 0..N-1.
type Graph struct {
	Edges map[int][]int
}

// Detect runs the Louvain method to find communities.
// This is a naive greedy modularity optimization implementation.
func Detect(g Graph) []Community {
	nodeCommunity := make(map[int]int)
	for n := range g.Edges {
		nodeCommunity[n] = n // Initially each node is its own community
	}

	improved := true
	for improved {
		improved = false
		for n := range g.Edges {
			currentComm := nodeCommunity[n]
			bestComm := currentComm
			bestGain := 0.0

			// Evaluate neighbors
			neighbors := g.Edges[n]
			for _, neighbor := range neighbors {
				targetComm := nodeCommunity[neighbor]
				if targetComm == currentComm {
					continue
				}

				// Calculate Modularity Gain (Delta Q).
				// Placeholder logic: just favor larger overlap?
				gain := 1.0 // Mock calculation
				if gain > bestGain {
					bestGain = gain
					bestComm = targetComm
				}
			}

			if bestComm != currentComm {
				nodeCommunity[n] = bestComm
				improved = true
			}
		}
	}

	// Group
	commMap := make(map[int][]int)
	for n, c := range nodeCommunity {
		commMap[c] = append(commMap[c], n)
	}

	res := make([]Community, 0, len(commMap))
	for _, nodes := range commMap {
		res = append(res, Community{Nodes: nodes})
	}
	return res
}

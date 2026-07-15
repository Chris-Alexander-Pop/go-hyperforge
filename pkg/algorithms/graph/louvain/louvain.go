package louvain

// Educational sketch only — see package doc.

// Community represents a set of nodes.
type Community struct {
	Nodes []int
}

// Graph is a simple undirected adjacency list for node IDs.
// Edges are treated as undirected with weight 1.
type Graph struct {
	Edges map[int][]int
}

const maxPasses = 64

// Detect runs a Louvain-style greedy modularity optimization (first phase only).
// Modularity gain uses the standard ΔQ formula for unweighted undirected graphs.
func Detect(g Graph) []Community {
	nodes := make([]int, 0, len(g.Edges))
	for n := range g.Edges {
		nodes = append(nodes, n)
	}
	if len(nodes) == 0 {
		return nil
	}

	degree := make(map[int]float64, len(nodes))
	var m float64
	seen := make(map[[2]int]struct{})
	for u, nbrs := range g.Edges {
		degree[u] = float64(len(nbrs))
		for _, v := range nbrs {
			a, b := u, v
			if a > b {
				a, b = b, a
			}
			key := [2]int{a, b}
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			m++
		}
	}
	if m == 0 {
		res := make([]Community, 0, len(nodes))
		for _, n := range nodes {
			res = append(res, Community{Nodes: []int{n}})
		}
		return res
	}

	nodeCommunity := make(map[int]int, len(nodes))
	for _, n := range nodes {
		nodeCommunity[n] = n
	}

	sigmaTot := make(map[int]float64, len(nodes))
	for _, n := range nodes {
		sigmaTot[n] = degree[n]
	}

	for pass := 0; pass < maxPasses; pass++ {
		moved := false
		for _, n := range nodes {
			currentComm := nodeCommunity[n]
			bestComm := currentComm
			bestGain := 0.0

			ki := degree[n]
			sigmaTot[currentComm] -= ki

			for _, targetComm := range uniqueNeighborCommunities(g.Edges[n], nodeCommunity, currentComm) {
				kiIn := linksToCommunity(n, targetComm, g.Edges, nodeCommunity)
				gain := modularityGain(m, ki, kiIn, sigmaTot[targetComm])
				if gain > bestGain {
					bestGain = gain
					bestComm = targetComm
				}
			}

			if bestComm != currentComm && bestGain > 1e-12 {
				nodeCommunity[n] = bestComm
				sigmaTot[bestComm] += ki
				moved = true
			} else {
				sigmaTot[currentComm] += ki
				nodeCommunity[n] = currentComm
			}
		}
		if !moved {
			break
		}
	}

	commMap := make(map[int][]int)
	for n, c := range nodeCommunity {
		commMap[c] = append(commMap[c], n)
	}

	res := make([]Community, 0, len(commMap))
	for _, ns := range commMap {
		res = append(res, Community{Nodes: ns})
	}
	return res
}

// modularityGain computes ΔQ for moving node i into community C (Blondel et al.):
//
//	ΔQ = ki_in/(2m) − Σtot·ki/(2m)²
func modularityGain(m, ki, kiIn, sigmaTot float64) float64 {
	if m == 0 {
		return 0
	}
	return kiIn/(2*m) - (sigmaTot*ki)/(4*m*m)
}

func linksToCommunity(n, targetComm int, edges map[int][]int, nodeCommunity map[int]int) float64 {
	var kiIn float64
	for _, v := range edges[n] {
		if nodeCommunity[v] == targetComm {
			kiIn++
		}
	}
	return kiIn
}

func uniqueNeighborCommunities(neighbors []int, nodeCommunity map[int]int, skip int) []int {
	seen := make(map[int]struct{})
	out := make([]int, 0)
	for _, v := range neighbors {
		c := nodeCommunity[v]
		if c == skip {
			continue
		}
		if _, ok := seen[c]; ok {
			continue
		}
		seen[c] = struct{}{}
		out = append(out, c)
	}
	return out
}

// Modularity computes Q = (1/(2m)) Σ_ij [A_ij − k_i k_j/(2m)] δ(c_i, c_j).
func Modularity(g Graph, communities []Community) float64 {
	nodeCommunity := make(map[int]int)
	nodes := make([]int, 0)
	for i, c := range communities {
		for _, n := range c.Nodes {
			nodeCommunity[n] = i
			nodes = append(nodes, n)
		}
	}

	degree := make(map[int]float64)
	adj := make(map[[2]int]float64)
	var m float64
	for u, nbrs := range g.Edges {
		degree[u] = float64(len(nbrs))
		for _, v := range nbrs {
			if u < v {
				adj[[2]int{u, v}] = 1
				m++
			}
		}
	}
	if m == 0 {
		return 0
	}

	var q float64
	for _, i := range nodes {
		for _, j := range nodes {
			if nodeCommunity[i] != nodeCommunity[j] {
				continue
			}
			a, b := i, j
			if a > b {
				a, b = b, a
			}
			A := 0.0
			if i != j {
				A = adj[[2]int{a, b}]
			}
			q += A - (degree[i]*degree[j])/(2*m)
		}
	}
	return q / (2 * m)
}

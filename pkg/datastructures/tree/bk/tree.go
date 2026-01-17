package bk

// Tree supports fuzzy string matching using Levenshtein distance.
type Tree struct {
	root *Node
}

type Node struct {
	word     string
	children map[int]*Node // distance -> node
}

func New() *Tree {
	return &Tree{}
}

func (t *Tree) Add(word string) {
	if t.root == nil {
		t.root = &Node{word: word, children: make(map[int]*Node)}
		return
	}

	curr := t.root
	for {
		dist := levenshtein(word, curr.word)
		if dist == 0 {
			return // Duplicate
		}

		child, exists := curr.children[dist]
		if !exists {
			curr.children[dist] = &Node{word: word, children: make(map[int]*Node)}
			return
		}
		curr = child
	}
}

// Search returns words within maxDist.
func (t *Tree) Search(query string, maxDist int) []string {
	var results []string
	if t.root == nil {
		return results
	}

	t.search(t.root, query, maxDist, &results)
	return results
}

func (t *Tree) search(n *Node, query string, maxDist int, results *[]string) {
	d := levenshtein(query, n.word)
	if d <= maxDist {
		*results = append(*results, n.word)
	}

	start := d - maxDist
	if start < 0 {
		start = 0
	}
	end := d + maxDist

	for i := start; i <= end; i++ {
		if child, ok := n.children[i]; ok {
			t.search(child, query, maxDist, results)
		}
	}
}

func levenshtein(a, b string) int {
	la, lb := len(a), len(b)
	d := make([][]int, la+1)
	for i := range d {
		d[i] = make([]int, lb+1)
	}
	for i := 0; i <= la; i++ {
		d[i][0] = i
	}
	for j := 0; j <= lb; j++ {
		d[0][j] = j
	}

	for i := 1; i <= la; i++ {
		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			d[i][j] = min(
				d[i-1][j]+1, // deletion
				min(
					d[i][j-1]+1,      // insertion
					d[i-1][j-1]+cost, // substitution
				),
			)
		}
	}
	return d[la][lb]
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

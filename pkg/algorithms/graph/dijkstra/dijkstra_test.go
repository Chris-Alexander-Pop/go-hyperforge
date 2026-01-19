package dijkstra

import (
	"reflect"
	"testing"
)

func TestShortestPath(t *testing.T) {
	// A --1--> B --2--> D
	// |        ^
	// 4        | 1
	// v        |
	// C --3----
	g := Graph{
		"A": {"B": 1, "C": 4},
		"B": {"D": 2},
		"C": {"B": 1},
		"D": {},
	}

	tests := []struct {
		name     string
		start    string
		end      string
		wantDist float64
		wantPath []string
	}{
		{"A to D", "A", "D", 3, []string{"A", "B", "D"}},
		{"A to C", "A", "C", 4, []string{"A", "C"}},
		{"A to B", "A", "B", 1, []string{"A", "B"}},
		{"C to D", "C", "D", 3, []string{"C", "B", "D"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShortestPath(g, tt.start, tt.end)
			if got == nil {
				t.Fatalf("ShortestPath() returned nil, want path")
			}
			if got.Distance != tt.wantDist {
				t.Errorf("ShortestPath().Distance = %v, want %v", got.Distance, tt.wantDist)
			}
			if !reflect.DeepEqual(got.Path, tt.wantPath) {
				t.Errorf("ShortestPath().Path = %v, want %v", got.Path, tt.wantPath)
			}
		})
	}
}

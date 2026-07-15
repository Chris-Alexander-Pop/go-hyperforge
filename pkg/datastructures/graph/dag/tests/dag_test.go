package dag_test

import (
	"errors"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/datastructures/graph/dag"
)

func TestDAG_TopologicalSort(t *testing.T) {
	g := dag.New[string]()
	g.AddNode("A", "a")
	g.AddNode("B", "b")
	g.AddNode("C", "c")

	if err := g.AddEdge("A", "B"); err != nil {
		t.Fatal(err)
	}
	if err := g.AddEdge("B", "C"); err != nil {
		t.Fatal(err)
	}

	order, err := g.TopologicalSort()
	if err != nil {
		t.Fatal(err)
	}
	pos := map[string]int{}
	for i, id := range order {
		pos[id] = i
	}
	if pos["A"] > pos["B"] || pos["B"] > pos["C"] {
		t.Fatalf("invalid topo order %v", order)
	}
}

func TestDAG_CycleRejected(t *testing.T) {
	g := dag.New[int]()
	g.AddNode("A", 1)
	g.AddNode("B", 2)
	if err := g.AddEdge("A", "B"); err != nil {
		t.Fatal(err)
	}
	if err := g.AddEdge("B", "A"); !errors.Is(err, dag.ErrCycleDetected) {
		t.Fatalf("cycle err=%v want ErrCycleDetected", err)
	}
}

func TestDAG_SelfLoopRejected(t *testing.T) {
	g := dag.New[int]()
	g.AddNode("A", 1)
	if err := g.AddEdge("A", "A"); !errors.Is(err, dag.ErrCycleDetected) {
		t.Fatalf("self-loop err=%v want ErrCycleDetected", err)
	}
}

func TestDAG_MissingNode(t *testing.T) {
	g := dag.New[int]()
	g.AddNode("A", 1)
	if err := g.AddEdge("A", "missing"); !errors.Is(err, dag.ErrNodeNotFound) {
		t.Fatalf("err=%v want ErrNodeNotFound", err)
	}
}

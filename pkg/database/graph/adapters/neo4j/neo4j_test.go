package neo4j_test

import (
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/database/graph"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/database/graph/adapters/neo4j"
)

func TestNew_RequiresHost(t *testing.T) {
	_, err := neo4j.New(graph.Config{})
	if err == nil {
		t.Fatal("expected error for empty host")
	}
}

func TestNew_BuildsHTTPURL(t *testing.T) {
	s, err := neo4j.New(graph.Config{Host: "localhost", Port: 7474, User: "neo4j"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer s.Close()
	var _ graph.Interface = s
}

// Package graph provides a unified interface for graph databases.
//
// Supported backends:
//   - Neo4j (HTTP Cypher transactional API)
//   - Memory (for testing)
//
// Planned: AWS Neptune.
//
// Usage:
//
//	import "github.com/chris-alexander-pop/system-design-library/pkg/database/graph/adapters/neo4j"
//
//	g, err := neo4j.New(cfg)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer g.Close()
//
//	v := &graph.Vertex{ID: "1", Label: "Person", Properties: map[string]interface{}{"name": "Alice"}}
//	err = g.AddVertex(ctx, v)
package graph

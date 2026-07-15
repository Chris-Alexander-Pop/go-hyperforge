package weaviate_test

import (
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/database/vector"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/database/vector/adapters/weaviate"
)

func TestNew_RequiresHost(t *testing.T) {
	_, err := weaviate.New(vector.Config{})
	if err == nil {
		t.Fatal("expected error for empty host")
	}
}

func TestNew_DefaultsClass(t *testing.T) {
	s, err := weaviate.New(vector.Config{Host: "http://localhost:8080"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer s.Close()
	var _ vector.Store = s
}

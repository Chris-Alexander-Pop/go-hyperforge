package feature_test

import (
	"context"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/ai/ml/feature"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/ai/ml/feature/adapters/memory"
)

func TestInstrumentedMemoryFeatures(t *testing.T) {
	ctx := context.Background()
	store := feature.NewInstrumentedStore(memory.New())
	if err := store.CreateFeatureGroup(ctx, &feature.FeatureGroup{
		Name:      "users",
		EntityKey: "user_id",
		Features:  []feature.FeatureDefinition{{Name: "age", Type: feature.FeatureTypeInt}},
	}); err != nil {
		t.Fatalf("CreateFeatureGroup: %v", err)
	}
	if err := store.IngestFeatures(ctx, "users", []feature.FeatureVector{
		{EntityKey: "u1", Features: map[string]interface{}{"age": 30}},
	}); err != nil {
		t.Fatalf("Ingest: %v", err)
	}
	vecs, err := store.GetOnlineFeatures(ctx, "users", []string{"u1"}, []string{"age"})
	if err != nil {
		t.Fatalf("GetOnline: %v", err)
	}
	if len(vecs) != 1 || vecs[0].Features["age"] != 30 {
		t.Fatalf("unexpected vectors %+v", vecs)
	}
}

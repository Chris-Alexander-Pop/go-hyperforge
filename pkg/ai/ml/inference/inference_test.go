package inference_test

import (
	"context"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/ai/ml/inference"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/ai/ml/inference/adapters/memory"
)

func TestInstrumentedMemoryPredict(t *testing.T) {
	ctx := context.Background()
	srv := inference.NewInstrumentedServer(memory.New())
	_, err := srv.LoadModel(ctx, inference.Config{Name: "m1", ModelType: inference.ModelTypeONNX})
	if err != nil {
		t.Fatalf("LoadModel: %v", err)
	}
	resp, err := srv.Predict(ctx, &inference.PredictRequest{ModelName: "m1"})
	if err != nil {
		t.Fatalf("Predict: %v", err)
	}
	if resp.ModelName != "m1" {
		t.Fatalf("unexpected resp %+v", resp)
	}
	h, err := srv.Health(ctx)
	if err != nil || !h.Healthy || h.ModelsLoaded != 1 {
		t.Fatalf("Health: %+v %v", h, err)
	}
}

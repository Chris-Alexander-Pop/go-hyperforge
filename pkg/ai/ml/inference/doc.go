// Package inference provides model serving for ML inference.
//
// Usage:
//
//	import (
//	    "github.com/chris-alexander-pop/go-hyperforge/pkg/ai/ml/inference"
//	    "github.com/chris-alexander-pop/go-hyperforge/pkg/ai/ml/inference/adapters/memory"
//	)
//
//	server := inference.NewInstrumentedServer(memory.New())
//	result, err := server.Predict(ctx, input)
package inference

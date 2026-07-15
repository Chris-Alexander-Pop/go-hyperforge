// Package feature provides a feature store client for ML features.
//
// Supports feature retrieval for training and inference.
//
// Usage:
//
//	import (
//	    "github.com/chris-alexander-pop/go-hyperforge/pkg/ai/ml/feature"
//	    "github.com/chris-alexander-pop/go-hyperforge/pkg/ai/ml/feature/adapters/memory"
//	)
//
//	store := feature.NewInstrumentedStore(memory.New())
//	features, err := store.GetOnlineFeatures(ctx, "user-features", entityKeys, nil)
package feature

// Package memory provides an in-memory FeatureStore for tests and local use.
package memory

import (
	"github.com/chris-alexander-pop/go-hyperforge/pkg/ai/ml/feature"
)

// New returns an in-memory feature.FeatureStore.
func New() feature.FeatureStore {
	return feature.New(feature.Config{Backend: "memory"})
}

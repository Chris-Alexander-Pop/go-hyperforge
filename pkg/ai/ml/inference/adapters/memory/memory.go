// Package memory provides an in-memory InferenceServer for tests and local use.
package memory

import (
	"github.com/chris-alexander-pop/go-hyperforge/pkg/ai/ml/inference"
)

// New returns an in-memory inference.InferenceServer.
func New() inference.InferenceServer {
	return inference.NewMemoryServer()
}

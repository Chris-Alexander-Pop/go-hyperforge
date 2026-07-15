package memory

import (
	"context"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/data/bigdata"
)

type Adapter struct {
	// MockStorage could be here
}

func New() *Adapter {
	return &Adapter{}
}

func (a *Adapter) Query(ctx context.Context, query string, args ...interface{}) (*bigdata.Result, error) {
	// Mock implementation: returns empty result or could parse simple mock SQL
	return &bigdata.Result{
		Rows: []map[string]interface{}{
			{"id": 1, "name": "mock"},
		},
		Metadata: map[string]interface{}{"source": "memory"},
	}, nil
}

func (a *Adapter) Close() error {
	return nil
}

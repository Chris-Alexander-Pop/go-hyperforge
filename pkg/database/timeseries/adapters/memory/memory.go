package memory

import (
	"context"
	"fmt"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/concurrency"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/database/timeseries"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
)

// Timeseries implements an in-memory timeseries database.
type Timeseries struct {
	points []*timeseries.Point
	mu     *concurrency.SmartRWMutex
}

// New creates a new in-memory timeseries database.
func New() *Timeseries {
	return &Timeseries{
		points: make([]*timeseries.Point, 0),
		mu:     concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "memory-timeseries"}),
	}
}

// Write writes a single point to the database.
func (t *Timeseries) Write(ctx context.Context, point *timeseries.Point) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Clone the point to avoid mutation issues
	t.points = append(t.points, clonePoint(point))
	return nil
}

// WriteBatch writes a batch of points to the database.
func (t *Timeseries) WriteBatch(ctx context.Context, points []*timeseries.Point) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	for _, p := range points {
		t.points = append(t.points, clonePoint(p))
	}
	return nil
}

// Query executes a simple measurement-based query.
// Supported query format: "measurement_name" (returns all points for that measurement)
func (t *Timeseries) Query(ctx context.Context, query string) ([]*timeseries.Point, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var results []*timeseries.Point
	for _, p := range t.points {
		if p.Measurement == query {
			results = append(results, clonePoint(p))
		}
	}

	if len(results) == 0 {
		return nil, errors.NotFound(fmt.Sprintf("no points found for measurement: %s", query), nil)
	}

	return results, nil
}

// Close clears the in-memory store.
func (t *Timeseries) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.points = nil
	return nil
}

// clonePoint creates a deep copy of a point.
func clonePoint(p *timeseries.Point) *timeseries.Point {
	newP := &timeseries.Point{
		Measurement: p.Measurement,
		Time:        p.Time,
		Tags:        make(map[string]string),
		Fields:      make(map[string]interface{}),
	}
	for k, v := range p.Tags {
		newP.Tags[k] = v
	}
	for k, v := range p.Fields {
		newP.Fields[k] = v
	}
	return newP
}

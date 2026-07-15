package memory_test

import (
	"context"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/database/timeseries/adapters/memory"
	"testing"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/database/timeseries"
	"github.com/stretchr/testify/assert"
)

func TestMemoryTimeseries(t *testing.T) {
	ts := memory.New()
	ctx := context.Background()

	p1 := &timeseries.Point{
		Measurement: "cpu",
		Tags:        map[string]string{"host": "server1"},
		Fields:      map[string]interface{}{"value": 90.0},
		Time:        time.Now(),
	}

	// Test Write
	err := ts.Write(ctx, p1)
	assert.NoError(t, err)

	// Test Query
	results, err := ts.Query(ctx, "cpu")
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "server1", results[0].Tags["host"])

	// Test WriteBatch
	p2 := &timeseries.Point{
		Measurement: "memory",
		Tags:        map[string]string{"host": "server1"},
		Fields:      map[string]interface{}{"value": 1024},
		Time:        time.Now(),
	}
	p3 := &timeseries.Point{
		Measurement: "cpu",
		Tags:        map[string]string{"host": "server2"},
		Fields:      map[string]interface{}{"value": 50.0},
		Time:        time.Now(),
	}

	err = ts.WriteBatch(ctx, []*timeseries.Point{p2, p3})
	assert.NoError(t, err)

	// Test Query again
	results, err = ts.Query(ctx, "cpu")
	assert.NoError(t, err)
	assert.Len(t, results, 2)

	// Test NotFound
	_, err = ts.Query(ctx, "unknown")
	assert.Error(t, err)

	// Test Close
	err = ts.Close()
	assert.NoError(t, err)
}

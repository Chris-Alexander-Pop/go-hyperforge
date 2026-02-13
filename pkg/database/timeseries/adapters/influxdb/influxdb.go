package influxdb

import (
	"context"
	"fmt"

	"github.com/chris-alexander-pop/system-design-library/pkg/database/timeseries"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
)

// NOTE: This implementation assumes the InfluxDB v2 Go client is available.
// If not, please run: go get github.com/influxdata/influxdb-client-go/v2/...

// Adapter implements the Timeseries interface for InfluxDB.
type Adapter struct {
	client   influxdb2.Client
	writeAPI api.WriteAPIBlocking
	queryAPI api.QueryAPI
	org      string
	bucket   string
}

// New creates a new InfluxDB adapter.
func New(cfg timeseries.Config) (*Adapter, error) {
	url := fmt.Sprintf("http://%s:%d", cfg.Host, cfg.Port)
	client := influxdb2.NewClient(url, cfg.Password) // Password is typically the token in v2

	// Use blocking write API for simplicity and errors return.
	// For high throughput, non-blocking API with async error handling might be better.
	writeAPI := client.WriteAPIBlocking(cfg.Org, cfg.Database)
	queryAPI := client.QueryAPI(cfg.Org)

	return &Adapter{
		client:   client,
		writeAPI: writeAPI,
		queryAPI: queryAPI,
		org:      cfg.Org,
		bucket:   cfg.Database,
	}, nil
}

// Write writes a single point to InfluxDB.
func (a *Adapter) Write(ctx context.Context, point *timeseries.Point) error {
	p := convertPoint(point)
	if err := a.writeAPI.WritePoint(ctx, p); err != nil {
		return errors.Internal("failed to write point to influxdb", err)
	}
	return nil
}

// WriteBatch writes a batch of points to InfluxDB.
func (a *Adapter) WriteBatch(ctx context.Context, points []*timeseries.Point) error {
	if len(points) == 0 {
		return nil
	}

	influxPoints := make([]*write.Point, 0, len(points))
	for _, p := range points {
		influxPoints = append(influxPoints, convertPoint(p))
	}

	if err := a.writeAPI.WritePoint(ctx, influxPoints...); err != nil {
		return errors.Internal("failed to write batch to influxdb", err)
	}
	return nil
}

// Query executes a Flux query against InfluxDB.
func (a *Adapter) Query(ctx context.Context, query string) ([]*timeseries.Point, error) {
	result, err := a.queryAPI.Query(ctx, query)
	if err != nil {
		return nil, errors.Internal("failed to execute influxdb query", err)
	}
	defer result.Close()

	var points []*timeseries.Point

	// Flux query results come as a stream of records.
	// Grouping them back into "Points" can be tricky if fields are split across records.
	// Simplified assumption: each record represents a field of a point.
	// Real-world implementation needs robust result parsing/pivoting.

	for result.Next() {
		record := result.Record()

		// This simplified mapping creates one point per field record,
		// which is essentially how InfluxDB stores data (series).
		// A more complex aggregation would be needed to reconstruct full rows.

		p := &timeseries.Point{
			Measurement: record.Measurement(),
			Tags:        make(map[string]string),
			Fields:      make(map[string]interface{}),
			Time:        record.Time(),
		}

		for k, v := range record.Values() {
			if k == "_value" {
				p.Fields[record.Field()] = v
			} else if k != "_time" && k != "_start" && k != "_stop" && k != "_measurement" && k != "_field" {
				// Treat others as tags/metadata
				if strVal, ok := v.(string); ok {
					p.Tags[k] = strVal
				}
			}
		}
		points = append(points, p)
	}

	if result.Err() != nil {
		return nil, errors.Internal("error iterating query results", result.Err())
	}

	return points, nil
}

// Close closes the InfluxDB client.
func (a *Adapter) Close() error {
	a.client.Close()
	return nil
}

// convertPoint converts a generic Point to an InfluxDB Point.
func convertPoint(p *timeseries.Point) *write.Point {
	return influxdb2.NewPoint(
		p.Measurement,
		p.Tags,
		p.Fields,
		p.Time,
	)
}

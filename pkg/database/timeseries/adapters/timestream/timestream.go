package timestream

import (
	"context"
	"fmt"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/timestreamquery"
	"github.com/aws/aws-sdk-go-v2/service/timestreamwrite"
	"github.com/aws/aws-sdk-go-v2/service/timestreamwrite/types"
	"github.com/chris-alexander-pop/system-design-library/pkg/database/timeseries"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
)

// NOTE: This implementation assumes the AWS SDK v2 is available.
// If not, please run: go get github.com/aws/aws-sdk-go-v2/...

// Adapter implements the Timeseries interface for AWS Timestream.
type Adapter struct {
	writeClient *timestreamwrite.Client
	queryClient *timestreamquery.Client
	database    string
}

// New creates a new AWS Timestream adapter.
func New(cfg timeseries.Config) (*Adapter, error) {
	// Load AWS config (credentials from env/profile)
	awsCfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(cfg.Region))
	if err != nil {
		return nil, errors.Internal("failed to load aws config", err)
	}

	return &Adapter{
		writeClient: timestreamwrite.NewFromConfig(awsCfg),
		queryClient: timestreamquery.NewFromConfig(awsCfg),
		database:    cfg.Database,
	}, nil
}

// Write writes a single point to Timestream.
func (a *Adapter) Write(ctx context.Context, point *timeseries.Point) error {
	record := convertPoint(point)

	_, err := a.writeClient.WriteRecords(ctx, &timestreamwrite.WriteRecordsInput{
		DatabaseName: aws.String(a.database),
		TableName:    aws.String(point.Measurement),
		Records:      []types.Record{record},
	})

	if err != nil {
		return errors.Internal("failed to write record to timestream", err)
	}
	return nil
}

// WriteBatch writes a batch of points to Timestream.
// Note: Timestream likely has limits on batch size (e.g. 100 records).
// Production impl should handle chunking.
func (a *Adapter) WriteBatch(ctx context.Context, points []*timeseries.Point) error {
	if len(points) == 0 {
		return nil
	}

	// Group by measurement/table as Timestream writes are per-table
	grouped := make(map[string][]types.Record)
	for _, p := range points {
		grouped[p.Measurement] = append(grouped[p.Measurement], convertPoint(p))
	}

	for table, records := range grouped {
		// Basic chunking (Timestream max is 100 records per call usually)
		const maxBatch = 100
		for i := 0; i < len(records); i += maxBatch {
			end := i + maxBatch
			if end > len(records) {
				end = len(records)
			}

			_, err := a.writeClient.WriteRecords(ctx, &timestreamwrite.WriteRecordsInput{
				DatabaseName: aws.String(a.database),
				TableName:    aws.String(table),
				Records:      records[i:end],
			})
			if err != nil {
				return errors.Internal(fmt.Sprintf("failed to write batch to table %s", table), err)
			}
		}
	}

	return nil
}

// Query executes a SQL query against Timestream.
func (a *Adapter) Query(ctx context.Context, query string) ([]*timeseries.Point, error) {
	output, err := a.queryClient.Query(ctx, &timestreamquery.QueryInput{
		QueryString: aws.String(query),
	})
	if err != nil {
		return nil, errors.Internal("failed to execute timestream query", err)
	}

	// Parsing Timestream results is complex due to dynamic schema.
	// This is a simplified example returning generic points.
	results := make([]*timeseries.Point, len(output.Rows))
	for i, row := range output.Rows {
		p := &timeseries.Point{
			Fields: make(map[string]interface{}),
		}
		for j, datum := range row.Data {
			colInfo := output.ColumnInfo[j]
			name := aws.ToString(colInfo.Name)

			if datum.ScalarValue != nil {
				p.Fields[name] = aws.ToString(datum.ScalarValue)
			}
			// Handling time column specifically requires known name or type check
		}
		results[i] = p
	}

	return results, nil
}

// Close is a no-op for AWS clients as they use http.Client.
func (a *Adapter) Close() error {
	return nil
}

// convertPoint converts a generic Point to a Timestream Record.
func convertPoint(p *timeseries.Point) types.Record {
	dimensions := make([]types.Dimension, 0, len(p.Tags))
	for k, v := range p.Tags {
		dimensions = append(dimensions, types.Dimension{
			Name:  aws.String(k),
			Value: aws.String(v),
		})
	}

	// Prepare measure value. Timestream supports Multi-measure records.
	// For simplicity, we handle simple multi-measure records here using MeasureValues.
	measureValues := make([]types.MeasureValue, 0, len(p.Fields))
	for k, v := range p.Fields {
		valStr := fmt.Sprintf("%v", v)
		valType := types.MeasureValueTypeVarchar // Default to string

		switch v.(type) {
		case float64, float32:
			valType = types.MeasureValueTypeDouble
		case int, int64:
			valType = types.MeasureValueTypeBigint
		case bool:
			valType = types.MeasureValueTypeBoolean
		}

		measureValues = append(measureValues, types.MeasureValue{
			Name:  aws.String(k),
			Value: aws.String(valStr),
			Type:  valType,
		})
	}

	// Use timestamp in milliseconds
	timeVal := strconv.FormatInt(p.Time.UnixMilli(), 10)

	return types.Record{
		Dimensions:       dimensions,
		MeasureName:      aws.String("metrics"), // Common measure name for multi-measure records
		MeasureValues:    measureValues,
		MeasureValueType: types.MeasureValueTypeMulti,
		Time:             aws.String(timeVal),
		TimeUnit:         types.TimeUnitMilliseconds,
	}
}

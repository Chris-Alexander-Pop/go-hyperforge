package bigquery

import (
	"context"

	"cloud.google.com/go/bigquery"
	"github.com/chris-alexander-pop/system-design-library/pkg/bigdata"
	"google.golang.org/api/iterator"
)

type Adapter struct {
	client *bigquery.Client
}

func New(ctx context.Context, projectID string) (*Adapter, error) {
	client, err := bigquery.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return &Adapter{client: client}, nil
}

func (a *Adapter) Query(ctx context.Context, q string, args ...interface{}) (*bigdata.Result, error) {
	query := a.client.Query(q)
	// Bind args generic mapping omitted for brevity, usually assumes internal params

	it, err := query.Read(ctx)
	if err != nil {
		return nil, err
	}

	var rows []map[string]interface{}
	for {
		var row map[string]bigquery.Value
		err := it.Next(&row)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		// Convert map[string]bigquery.Value to map[string]interface{}
		convertedRow := make(map[string]interface{}, len(row))
		for k, v := range row {
			convertedRow[k] = v
		}
		rows = append(rows, convertedRow)
	}

	return &bigdata.Result{Rows: rows}, nil
}

func (a *Adapter) Close() error {
	return a.client.Close()
}

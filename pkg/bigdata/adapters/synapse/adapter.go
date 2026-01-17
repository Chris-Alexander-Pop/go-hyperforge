package synapse

import (
	"context"
	"database/sql"

	"github.com/chris-alexander-pop/system-design-library/pkg/bigdata"
	_ "github.com/microsoft/go-mssqldb" // SQL Server Driver
)

type Adapter struct {
	db *sql.DB
}

func New(dsn string) (*Adapter, error) {
	db, err := sql.Open("sqlserver", dsn)
	if err != nil {
		return nil, err
	}
	return &Adapter{db: db}, nil
}

func (a *Adapter) Query(ctx context.Context, query string, args ...interface{}) (*bigdata.Result, error) {
	rows, err := a.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var results []map[string]interface{}
	for rows.Next() {
		// Generic reading into interface{}
		values := make([]interface{}, len(cols))
		valuePtrs := make([]interface{}, len(cols))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		row := make(map[string]interface{})
		for i, col := range cols {
			val := values[i]
			if b, ok := val.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}
		results = append(results, row)
	}

	return &bigdata.Result{Rows: results}, nil
}

func (a *Adapter) Close() error {
	return a.db.Close()
}
